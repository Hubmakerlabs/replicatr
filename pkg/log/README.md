# log

This is a very simple, but practical library for logging in applications. Its
main feature is printing source code locations to make debugging easier.

## dev machine filesystem hygiene

It is advisable to take care with the location you store your source code from which you build released binaries that it not leak information.

My personal recommendation is to follow the old "GOPATH" structure.

Make a folder at the root of your user home directory `src`, and then create directories for each git hosting URL, such as `github.com` and then place your repositories in a directory tree that matches the URLs, as with this one `src/mleku.online/git/log`. Then, as root, create a folder in the root with the path `/src` and then link the root URL directories down from your home folder to the `/src` in the root.

In this way, the log prints will then show (as my releases for this one) `/src/mleku.online/git/log/...` for code locations, and you will be able to click them in the terminal of editors like Goland and jump to the place where the log statement is found.

## space separated line structured

The prints are structured to be easily parsed into positional fields, space separated, with the log text field wrapped in backticks, which can contain single and double quotes and line breaks without disrupting the parsing - all backticks ( ` ) are replaced with single quotes (  '  ) giving maxmimum options for writing log texts with this one exception:

```
1701771065.141435472 TRC `testing log level Trace` /src/mleku.online/git/log/log_test.go:16
1701771065.141484431 DBG `testing log level Debug` /src/mleku.online/git/log/log_test.go:17
1701771065.141505593 INF `testing log level Info` /src/mleku.online/git/log/log_test.go:18
1701771065.141524869 WRN `testing log level Warn` /src/mleku.online/git/log/log_test.go:19
1701771065.141543936 ERR `testing log level Error` /src/mleku.online/git/log/log_test.go:20
1701771065.141562793 FTL `testing log level Fatal` /src/mleku.online/git/log/log_test.go:21
1701771065.141578647 ERR `CHECK: dummy error as error` /src/mleku.online/git/log/log_test.go:22
1701771065.141599181 INF `CHECK: dummy information check` /src/mleku.online/git/log/log_test.go:23
1701771065.141617619 INF `'backtick wrapped string'
(*testing.T)(0xc000007a00)({
 common: (testing.common) {
  mu: (sync.RWMutex) {
   w: (sync.Mutex) {
    state: (int32) 0,
    sema: (uint32) 0
   },
   writerSem: (uint32) 0,
   readerSem: (uint32) 0,
   readerCount: (atomic.Int32) {
    _: (atomic.noCopy) {
    },
    v: (int32) 0
   },
   readerWait: (atomic.Int32) {
    _: (atomic.noCopy) {
    },
    v: (int32) 0
   }
  },
  output: ([]uint8) <nil>,
  w: (testing.indenter) {
   c: (*testing.common)(0xc000007a00)(<already shown>)
  },
  ran: (bool) false,
  failed: (bool) false,
  skipped: (bool) false,
  done: (bool) false,
  helperPCs: (map[uintptr]struct {}) <nil>,
  helperNames: (map[string]struct {}) <nil>,
  cleanups: ([]func()) <nil>,
  cleanupName: (string) "",
  cleanupPc: ([]uintptr) <nil>,
  finished: (bool) false,
  inFuzzFn: (bool) false,
  chatty: (*testing.chattyPrinter)(0xc000110630)({
   w: (*os.File)(0xc000052020)({
    file: (*os.file)(0xc0000700c0)({
     pfd: (poll.FD) {
      fdmu: (poll.fdMutex) {
       state: (uint64) 0,
       rsema: (uint32) 0,
       wsema: (uint32) 0
      },
      Sysfd: (int) 1,
      SysFile: (poll.SysFile) {
       iovecs: (*[]syscall.Iovec)(<nil>)
      },
      pd: (poll.pollDesc) {
       runtimeCtx: (uintptr) <nil>
      },
      csema: (uint32) 0,
      isBlocking: (uint32) 1,
      IsStream: (bool) true,
      ZeroReadIsEOF: (bool) true,
      isFile: (bool) true
     },
     name: (string) (len=11) "/dev/stdout",
     dirinfo: (*os.dirInfo)(<nil>),
     nonblock: (bool) false,
     stdoutOrErr: (bool) true,
     appendMode: (bool) false
    })
   }),
   lastNameMu: (sync.Mutex) {
    state: (int32) 0,
    sema: (uint32) 0
   },
   lastName: (string) (len=13) "TestGetLogger",
   json: (bool) false
  }),
  bench: (bool) false,
  hasSub: (atomic.Bool) {
   _: (atomic.noCopy) {
   },
   v: (uint32) 0
  },
  cleanupStarted: (atomic.Bool) {
   _: (atomic.noCopy) {
   },
   v: (uint32) 0
  },
  raceErrors: (int) 0,
  runner: (string) (len=15) "testing.tRunner",
  isParallel: (bool) false,
  parent: (*testing.common)(0xc000007860)({
   mu: (sync.RWMutex) {
    w: (sync.Mutex) {
     state: (int32) 0,
     sema: (uint32) 0
    },
    writerSem: (uint32) 0,
    readerSem: (uint32) 0,
    readerCount: (atomic.Int32) {
     _: (atomic.noCopy) {
     },
     v: (int32) 0
    },
    readerWait: (atomic.Int32) {
     _: (atomic.noCopy) {
     },
     v: (int32) 0
    }
   },
   output: ([]uint8) <nil>,
   w: (*os.File)(0xc000052020)({
    file: (*os.file)(0xc0000700c0)({
     pfd: (poll.FD) {
      fdmu: (poll.fdMutex) {
       state: (uint64) 0,
       rsema: (uint32) 0,
       wsema: (uint32) 0
      },
      Sysfd: (int) 1,
      SysFile: (poll.SysFile) {
       iovecs: (*[]syscall.Iovec)(<nil>)
      },
      pd: (poll.pollDesc) {
       runtimeCtx: (uintptr) <nil>
      },
      csema: (uint32) 0,
      isBlocking: (uint32) 1,
      IsStream: (bool) true,
      ZeroReadIsEOF: (bool) true,
      isFile: (bool) true
     },
     name: (string) (len=11) "/dev/stdout",
     dirinfo: (*os.dirInfo)(<nil>),
     nonblock: (bool) false,
     stdoutOrErr: (bool) true,
     appendMode: (bool) false
    })
   }),
   ran: (bool) false,
   failed: (bool) false,
   skipped: (bool) false,
   done: (bool) false,
   helperPCs: (map[uintptr]struct {}) <nil>,
   helperNames: (map[string]struct {}) <nil>,
   cleanups: ([]func()) <nil>,
   cleanupName: (string) "",
   cleanupPc: ([]uintptr) <nil>,
   finished: (bool) false,
   inFuzzFn: (bool) false,
   chatty: (*testing.chattyPrinter)(0xc000110630)({
    w: (*os.File)(0xc000052020)({
     file: (*os.file)(0xc0000700c0)({
      pfd: (poll.FD) {
       fdmu: (poll.fdMutex) {
        state: (uint64) 0,
        rsema: (uint32) 0,
        wsema: (uint32) 0
       },
       Sysfd: (int) 1,
       SysFile: (poll.SysFile) {
        iovecs: (*[]syscall.Iovec)(<nil>)
       },
       pd: (poll.pollDesc) {
        runtimeCtx: (uintptr) <nil>
       },
       csema: (uint32) 0,
       isBlocking: (uint32) 1,
       IsStream: (bool) true,
       ZeroReadIsEOF: (bool) true,
       isFile: (bool) true
      },
      name: (string) (len=11) "/dev/stdout",
      dirinfo: (*os.dirInfo)(<nil>),
      nonblock: (bool) false,
      stdoutOrErr: (bool) true,
      appendMode: (bool) false
     })
    }),
    lastNameMu: (sync.Mutex) {
     state: (int32) 0,
     sema: (uint32) 0
    },
    lastName: (string) (len=13) "TestGetLogger",
    json: (bool) false
   }),
   bench: (bool) false,
   hasSub: (atomic.Bool) {
    _: (atomic.noCopy) {
    },
    v: (uint32) 1
   },
   cleanupStarted: (atomic.Bool) {
    _: (atomic.noCopy) {
    },
    v: (uint32) 0
   },
   raceErrors: (int) 0,
   runner: (string) (len=15) "testing.tRunner",
   isParallel: (bool) false,
   parent: (*testing.common)(<nil>),
   level: (int) 0,
   creator: ([]uintptr) <nil>,
   name: (string) "",
   start: (time.Time) 2023-12-05 10:11:05.141341466 +0000 WET m=+0.000506001,
   duration: (time.Duration) 0s,
   barrier: (chan bool) 0xc00007a360,
   signal: (chan bool) (cap=1) 0xc0000282a0,
   sub: ([]*testing.T) <nil>,
   tempDirMu: (sync.Mutex) {
    state: (int32) 0,
    sema: (uint32) 0
   },
   tempDir: (string) "",
   tempDirErr: (error) <nil>,
   tempDirSeq: (int32) 0
  }),
  level: (int) 1,
  creator: ([]uintptr) (len=7 cap=50) {
   (uintptr) 0x4c5d5e,
   (uintptr) 0x4c2e7f,
   (uintptr) 0x4c5c45,
   (uintptr) 0x4c4636,
   (uintptr) 0x50a75c,
   (uintptr) 0x43925b,
   (uintptr) 0x468d61
  },
  name: (string) (len=13) "TestGetLogger",
  start: (time.Time) 2023-12-05 10:11:05.141424857 +0000 WET m=+0.000588414,
  duration: (time.Duration) 0s,
  barrier: (chan bool) 0xc00007a3c0,
  signal: (chan bool) (cap=1) 0xc000028310,
  sub: ([]*testing.T) <nil>,
  tempDirMu: (sync.Mutex) {
   state: (int32) 0,
   sema: (uint32) 0
  },
  tempDir: (string) "",
  tempDirErr: (error) <nil>,
  tempDirSeq: (int32) 0
 },
 isEnvSet: (bool) false,
 context: (*testing.testContext)(0xc000014280)({
  match: (*testing.matcher)(0xc00002c8c0)({
   filter: (testing.simpleMatch) (len=1 cap=1) {
    (string) (len=19) "^\\QTestGetLogger\\E$"
   },
   skip: (testing.alternationMatch) {
   },
   matchFunc: (func(string, string) (bool, error)) 0x4c8140,
   mu: (sync.Mutex) {
    state: (int32) 0,
    sema: (uint32) 0
   },
   subNames: (map[string]int32) {
   }
  }),
  deadline: (time.Time) 0001-01-01 00:00:00 +0000 UTC,
  isFuzzing: (bool) false,
  mu: (sync.Mutex) {
   state: (int32) 0,
   sema: (uint32) 0
  },
  startParallel: (chan bool) 0xc00007a300,
  running: (int) 1,
  numWaiting: (int) 0,
  maxParallel: (int) 8
 })
})
` /src/mleku.online/git/log/log_test.go:25
```

The timestamp is in seconds, with 9 decimal places to represent nanoseconds after the decimal point.

The format is designed for optimal readability by humans while maintaining readability by machines, unsurprisingly similar to Go syntax itself and for the same reasons.

This will also enable bulk capture and filtering of the logs. They are written by default to `os.Stderr` to appear as expected in the systemd journal if you run the application this way.

The writer is a singleton value (package local variable) so if you create a writer that logs to disk or streams the logs to a socket or whatever, in the one application the same writer will be used and funnel all the output through it.
