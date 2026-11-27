//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// stopSignals are the signals weft dev stops its children on and exits 0.
var stopSignals = []os.Signal{os.Interrupt}

const (
	// createSuspended is CREATE_SUSPENDED: the new process's main thread
	// does not run until ResumeThread is called, so AssignProcessToJobObject
	// puts the process in its Job Object before it can execute a single
	// instruction or spawn a child.
	createSuspended = 0x00000004
	// jobObjectBasicAccountingInformation is JobObjectBasicAccountingInformation (1).
	jobObjectBasicAccountingInformation = 1
	// jobObjectExtendedLimitInformation is JobObjectExtendedLimitInformation (9).
	jobObjectExtendedLimitInformation = 9
	// jobObjectLimitKillOnJobClose is JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE.
	jobObjectLimitKillOnJobClose = 0x00002000
	// th32csSnapThread is TH32CS_SNAPTHREAD.
	th32csSnapThread = 0x00000004
	// threadSuspendResume is THREAD_SUSPEND_RESUME.
	threadSuspendResume = 0x0002
	// processSetQuota is PROCESS_SET_QUOTA (required by AssignProcessToJobObject).
	processSetQuota = 0x0100
)

var (
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObjectW       = kernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObj   = kernel32.NewProc("SetInformationJobObject")
	procQueryInformationJobObj = kernel32.NewProc("QueryInformationJobObject")
	procAssignProcessToJob     = kernel32.NewProc("AssignProcessToJobObject")
	procTerminateJobObject     = kernel32.NewProc("TerminateJobObject")
	procThread32First          = kernel32.NewProc("Thread32First")
	procThread32Next           = kernel32.NewProc("Thread32Next")
	procOpenThread             = kernel32.NewProc("OpenThread")
	procResumeThread           = kernel32.NewProc("ResumeThread")
)

// jobObjectBasicLimitInformation is Windows' JOBOBJECT_BASIC_LIMIT_INFORMATION.
type jobObjectBasicLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

// ioCounters is Windows' IO_COUNTERS.
type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

// jobObjectExtendedLimitInfo is Windows' JOBOBJECT_EXTENDED_LIMIT_INFORMATION.
type jobObjectExtendedLimitInfo struct {
	BasicLimitInformation jobObjectBasicLimitInformation
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

// jobObjectBasicAccountingInfo is Windows' JOBOBJECT_BASIC_ACCOUNTING_INFORMATION.
type jobObjectBasicAccountingInfo struct {
	TotalUserTime             int64
	TotalKernelTime           int64
	ThisPeriodTotalUserTime   int64
	ThisPeriodTotalKernelTime int64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

// threadEntry32 is Windows' THREADENTRY32.
type threadEntry32 struct {
	Size           uint32
	Usage          uint32
	ThreadID       uint32
	OwnerProcessID uint32
	BasePri        int32
	DeltaPri       int32
	Flags          uint32
}

// Compile-time struct layout assertions for Windows amd64 and arm64 (64-bit).
var (
	_ [64 - unsafe.Sizeof(jobObjectBasicLimitInformation{})]byte
	_ [unsafe.Sizeof(jobObjectBasicLimitInformation{}) - 64]byte
	_ [144 - unsafe.Sizeof(jobObjectExtendedLimitInfo{})]byte
	_ [unsafe.Sizeof(jobObjectExtendedLimitInfo{}) - 144]byte
	_ [64 - unsafe.Offsetof(jobObjectExtendedLimitInfo{}.IoInfo)]byte
	_ [unsafe.Offsetof(jobObjectExtendedLimitInfo{}.IoInfo) - 64]byte
	_ [48 - unsafe.Sizeof(jobObjectBasicAccountingInfo{})]byte
	_ [unsafe.Sizeof(jobObjectBasicAccountingInfo{}) - 48]byte
	_ [40 - unsafe.Offsetof(jobObjectBasicAccountingInfo{}.ActiveProcesses)]byte
	_ [unsafe.Offsetof(jobObjectBasicAccountingInfo{}.ActiveProcesses) - 40]byte
	_ [28 - unsafe.Sizeof(threadEntry32{})]byte
	_ [unsafe.Sizeof(threadEntry32{}) - 28]byte
)

// callErr normalizes a failed LazyProc.Call error so an Errno(0) or nil error
// is never returned as "error 0" / nil.
func callErr(err error) error {
	if err == nil || errors.Is(err, syscall.Errno(0)) {
		return syscall.EINVAL
	}
	return err
}

// jobs holds the Job Object handle for each process weft started and has not
// let go of. A Job Object replaces PID pinning and taskkill /T: every process
// assigned to the job, and every child or grandchild it spawns, belongs to the
// job by kernel handle (never by PID, so PID reuse cannot reach another
// program, and an intermediate parent that exits early cannot orphan its
// children). Setting JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE also ensures that when
// the job handle is closed — including when weft dev exits, is killed hard or
// has its console closed — Windows terminates every remaining process in the
// job. mu guards the jobs map and each TerminateJobObject call; killTree polls
// jobGone outside mu before unpin closes the handle, which is safe because no
// two callers stop or unpin the same cmd concurrently.
var (
	mu   sync.Mutex
	jobs = map[*exec.Cmd]syscall.Handle{}
)

// newKillOnCloseJob creates an unnamed Job Object configured with
// JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE.
func newKillOnCloseJob() (syscall.Handle, error) {
	r1, _, err := procCreateJobObjectW.Call(0, 0)
	if r1 == 0 {
		return 0, callErr(err)
	}
	job := syscall.Handle(r1)
	var info jobObjectExtendedLimitInfo
	info.BasicLimitInformation.LimitFlags = jobObjectLimitKillOnJobClose
	r1, _, err = procSetInformationJobObj.Call(
		uintptr(job),
		jobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		unsafe.Sizeof(info),
	)
	if r1 == 0 {
		syscall.CloseHandle(job)
		return 0, callErr(err)
	}
	return job, nil
}

// assignToJob puts the process with pid into job.
func assignToJob(job syscall.Handle, pid int) error {
	h, err := syscall.OpenProcess(processSetQuota|syscall.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(h)
	r1, _, cErr := procAssignProcessToJob.Call(uintptr(job), uintptr(h))
	if r1 == 0 {
		return callErr(cErr)
	}
	return nil
}

// resumeProcessThreads resumes every thread of pid after it was started with
// CREATE_SUSPENDED.
func resumeProcessThreads(pid uint32) error {
	snap, err := syscall.CreateToolhelp32Snapshot(th32csSnapThread, 0)
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(snap)
	var te threadEntry32
	te.Size = uint32(unsafe.Sizeof(te))
	r1, _, cErr := procThread32First.Call(uintptr(snap), uintptr(unsafe.Pointer(&te)))
	if r1 == 0 {
		return callErr(cErr)
	}
	resumed := false
	for {
		if te.OwnerProcessID == pid {
			th, _, tErr := procOpenThread.Call(threadSuspendResume, 0, uintptr(te.ThreadID))
			if th == 0 {
				return callErr(tErr)
			}
			res, _, rErr := procResumeThread.Call(th)
			syscall.CloseHandle(syscall.Handle(th))
			if uint32(res) == ^uint32(0) {
				return callErr(rErr)
			}
			resumed = true
		}
		te.Size = uint32(unsafe.Sizeof(te))
		r1, _, _ = procThread32Next.Call(uintptr(snap), uintptr(unsafe.Pointer(&te)))
		if r1 == 0 {
			break
		}
	}
	if !resumed {
		return syscall.EINVAL
	}
	return nil
}

// startGroup starts cmd suspended, puts it in a Job Object of its own with
// JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE, and resumes its main thread, so no child
// can be spawned outside the job.
func startGroup(cmd *exec.Cmd) error {
	job, err := newKillOnCloseJob()
	if err != nil {
		return fmt.Errorf("%w: %v", errNoJob, err)
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	origFlags := cmd.SysProcAttr.CreationFlags
	cmd.SysProcAttr.CreationFlags = origFlags | createSuspended
	err = cmd.Start()
	cmd.SysProcAttr.CreationFlags = origFlags
	if err != nil {
		syscall.CloseHandle(job)
		return err
	}
	if err := assignToJob(job, cmd.Process.Pid); err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		syscall.CloseHandle(job)
		return fmt.Errorf("%w: %v", errNoJob, err)
	}
	mu.Lock()
	jobs[cmd] = job
	mu.Unlock()
	if err := resumeProcessThreads(uint32(cmd.Process.Pid)); err != nil {
		// Kill cmd directly before unpin closes the job handle: if the
		// caller's own process is also in this job (unexpected), TerminateJobObject
		// would terminate the caller itself.
		cmd.Process.Kill()
		cmd.Wait()
		mu.Lock()
		delete(jobs, cmd)
		mu.Unlock()
		syscall.CloseHandle(job)
		return fmt.Errorf("%w: %v", errNoJob, err)
	}
	return nil
}

// pin does nothing on Windows: every child is started with startGroup, which
// puts it in its Job Object while still suspended.
func pin(*exec.Cmd) {}

// unpin closes the Job Object handle for cmd, if any, and waits for its
// active process count to reach 0. Because the job carries
// JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE, closing its handle also terminates any
// grandchild still in the job after cmd itself has exited and been waited for.
func unpin(cmd *exec.Cmd) {
	mu.Lock()
	job, ok := jobs[cmd]
	if ok {
		delete(jobs, cmd)
	}
	mu.Unlock()
	if ok {
		procTerminateJobObject.Call(uintptr(job), 1)
		jobGone(job, time.Second)
		syscall.CloseHandle(job)
	}
}

// jobGone polls QueryInformationJobObject until ActiveProcesses in job is 0,
// for up to within.
func jobGone(job syscall.Handle, within time.Duration) bool {
	for deadline := time.Now().Add(within); ; time.Sleep(15 * time.Millisecond) {
		var info jobObjectBasicAccountingInfo
		r1, _, _ := procQueryInformationJobObj.Call(
			uintptr(job),
			jobObjectBasicAccountingInformation,
			uintptr(unsafe.Pointer(&info)),
			unsafe.Sizeof(info),
			0,
		)
		if r1 == 0 || info.ActiveProcesses == 0 {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
	}
}

// killTree terminates every process in cmd's Job Object with
// TerminateJobObject, such as the node that pnpm exec runs or the compilers go
// build runs, even if an intermediate parent has already exited, and waits up
// to 1 s for the job's processes to exit. When cmd has no job, or if
// TerminateJobObject fails, cmd itself is killed through its os.Process.
func killTree(cmd *exec.Cmd) {
	mu.Lock()
	job, ok := jobs[cmd]
	terminated := false
	if ok {
		r1, _, _ := procTerminateJobObject.Call(uintptr(job), 1)
		terminated = r1 != 0
	}
	mu.Unlock()
	if ok && terminated {
		jobGone(job, time.Second)
	}
	if (!ok || !terminated) && cmd.Process != nil {
		cmd.Process.Kill()
	}
}

// stopGroup kills cmd and what it started, waits up to 1 s for every process
// in the job to exit, and closes its Job Object handle.
func stopGroup(cmd *exec.Cmd) {
	killTree(cmd)
	unpin(cmd)
}

// interrupt kills cmd and what it started: Windows has no interrupt to send
// another process.
func interrupt(cmd *exec.Cmd) error {
	killTree(cmd)
	return nil
}

// setAside moves bin, a binary weft dev's Go server may be running from, to
// bin.old, so the next build can write bin: Windows lets a running binary be
// renamed, but not written over. The bin.old of the build before is no
// longer running by then, so it is replaced. A bin that is not there is
// left so.
func setAside(bin string) {
	os.Rename(bin, bin+".old")
}
