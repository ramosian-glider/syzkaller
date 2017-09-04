// Copyright 2015 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

// I heard you like shell...
//go:generate bash -c "echo -e '// AUTOGENERATED FROM executor/common.h\npackage csource\nvar commonHeader = `' > common.go; cat ../../executor/common.h | sed -e '/#include \"common_kvm_amd64.h\"/ {' -e 'r ../../executor/common_kvm_amd64.h' -e 'd' -e '}' - | sed -e '/#include \"common_kvm_arm64.h\"/ {' -e 'r ../../executor/common_kvm_arm64.h' -e 'd' -e '}' - | sed -e '/#include \"kvm.h\"/ {' -e 'r ../../executor/kvm.h' -e 'd' -e '}' - | sed -e '/#include \"kvm.S.h\"/ {' -e 'r ../../executor/kvm.S.h' -e 'd' -e '}' - | egrep -v '^[   ]*//' | sed '/^[ 	]*\\/\\/.*/d' | sed 's#[ 	]*//.*##g' >> common.go; echo '`' >> common.go"
//go:generate go fmt common.go

package csource

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unsafe"

	"github.com/google/syzkaller/prog"
	"github.com/google/syzkaller/sys"
)

type Options struct {
	Threaded bool
	Collide  bool
	Repeat   bool
	Procs    int
	Sandbox  string

	Fault     bool // inject fault into FaultCall/FaultNth
	FaultCall int
	FaultNth  int

	// These options allow for a more fine-tuned control over the generated C code.
	EnableTun  bool
	UseTmpDir  bool
	HandleSegv bool
	WaitRepeat bool
	Debug      bool

	// Generate code for use with repro package to prints log messages,
	// which allows to distinguish between a hang and an absent crash.
	Repro bool
}

// Check checks if the opts combination is valid or not.
// For example, Collide without Threaded is not valid.
// Invalid combinations must not be passed to Write.
func (opts Options) Check() error {
	if !opts.Threaded && opts.Collide {
		// Collide requires threaded.
		return errors.New("Collide without Threaded")
	}
	if !opts.Repeat && opts.Procs > 1 {
		// This does not affect generated code.
		return errors.New("Procs>1 without Repeat")
	}
	if opts.Sandbox == "namespace" && !opts.UseTmpDir {
		// This is borken and never worked.
		// This tries to create syz-tmp dir in cwd,
		// which will fail if procs>1 and on second run of the program.
		return errors.New("Sandbox=namespace without UseTmpDir")
	}
	return nil
}

func Write(p *prog.Prog, opts Options) ([]byte, error) {
	if err := opts.Check(); err != nil {
		return nil, fmt.Errorf("csource: invalid opts: %v", err)
	}
	exec := make([]byte, prog.ExecBufferSize)
	if err := p.SerializeForExec(exec, 0); err != nil {
		return nil, fmt.Errorf("failed to serialize program: %v", err)
	}
	w := new(bytes.Buffer)

	fmt.Fprint(w, "// autogenerated by syzkaller (http://github.com/google/syzkaller)\n\n")

	handled := make(map[string]uint64)
	for _, c := range p.Calls {
		handled[c.Meta.CallName] = c.Meta.NR
	}
	for name, nr := range handled {
		// Only generate defines for new syscalls (added after commit 8a1ab3155c2ac on 2012-10-04).
		// TODO: the syscall number 313 implies that we're dealing with linux/amd64.
		if nr >= 313 && !strings.HasPrefix(name, "syz_") {
			fmt.Fprintf(w, "#ifndef __NR_%v\n", name)
			fmt.Fprintf(w, "#define __NR_%v %v\n", name, nr)
			fmt.Fprintf(w, "#endif\n")
		}
	}
	fmt.Fprintf(w, "\n")

	hdr, err := preprocessCommonHeader(opts, handled, prog.RequiresBitmasks(p), prog.RequiresChecksums(p))
	if err != nil {
		return nil, err
	}
	fmt.Fprint(w, hdr)
	fmt.Fprint(w, "\n")

	calls, nvar := generateCalls(exec, opts)
	fmt.Fprintf(w, "long r[%v];\n", nvar)

	if !opts.Repeat {
		generateTestFunc(w, opts, calls, "loop")

		fmt.Fprint(w, "int main()\n{\n")
		if opts.HandleSegv {
			fmt.Fprintf(w, "\tinstall_segv_handler();\n")
		}
		if opts.UseTmpDir {
			fmt.Fprintf(w, "\tuse_temporary_dir();\n")
		}
		if opts.Sandbox != "" {
			fmt.Fprintf(w, "\tint pid = do_sandbox_%v(0, %v);\n", opts.Sandbox, opts.EnableTun)
			fmt.Fprint(w, "\tint status = 0;\n")
			fmt.Fprint(w, "\twhile (waitpid(pid, &status, __WALL) != pid) {}\n")
		} else {
			if opts.EnableTun {
				fmt.Fprintf(w, "\tsetup_tun(0, %v);\n", opts.EnableTun)
			}
			fmt.Fprint(w, "\tloop();\n")
		}
		fmt.Fprint(w, "\treturn 0;\n}\n")
	} else {
		generateTestFunc(w, opts, calls, "test")
		if opts.Procs <= 1 {
			fmt.Fprint(w, "int main()\n{\n")
			if opts.HandleSegv {
				fmt.Fprintf(w, "\tinstall_segv_handler();\n")
			}
			if opts.UseTmpDir {
				fmt.Fprintf(w, "\tuse_temporary_dir();\n")
			}
			if opts.Sandbox != "" {
				fmt.Fprintf(w, "\tint pid = do_sandbox_%v(0, %v);\n", opts.Sandbox, opts.EnableTun)
				fmt.Fprint(w, "\tint status = 0;\n")
				fmt.Fprint(w, "\twhile (waitpid(pid, &status, __WALL) != pid) {}\n")
			} else {
				if opts.EnableTun {
					fmt.Fprintf(w, "\tsetup_tun(0, %v);\n", opts.EnableTun)
				}
				fmt.Fprint(w, "\tloop();\n")
			}
			fmt.Fprint(w, "\treturn 0;\n}\n")
		} else {
			fmt.Fprint(w, "int main()\n{\n")
			fmt.Fprint(w, "\tint i;")
			fmt.Fprintf(w, "\tfor (i = 0; i < %v; i++) {\n", opts.Procs)
			fmt.Fprint(w, "\t\tif (fork() == 0) {\n")
			if opts.HandleSegv {
				fmt.Fprintf(w, "\t\t\tinstall_segv_handler();\n")
			}
			if opts.UseTmpDir {
				fmt.Fprintf(w, "\t\t\tuse_temporary_dir();\n")
			}
			if opts.Sandbox != "" {
				fmt.Fprintf(w, "\t\t\tint pid = do_sandbox_%v(i, %v);\n", opts.Sandbox, opts.EnableTun)
				fmt.Fprint(w, "\t\t\tint status = 0;\n")
				fmt.Fprint(w, "\t\t\twhile (waitpid(pid, &status, __WALL) != pid) {}\n")
			} else {
				if opts.EnableTun {
					fmt.Fprintf(w, "\t\t\tsetup_tun(i, %v);\n", opts.EnableTun)
				}
				fmt.Fprint(w, "\t\t\tloop();\n")
			}
			fmt.Fprint(w, "\t\t\treturn 0;\n")
			fmt.Fprint(w, "\t\t}\n")
			fmt.Fprint(w, "\t}\n")
			fmt.Fprint(w, "\tsleep(1000000);\n")
			fmt.Fprint(w, "\treturn 0;\n}\n")
		}
	}

	// Remove NONFAILING and debug calls.
	out0 := w.String()
	if !opts.HandleSegv {
		re := regexp.MustCompile(`\t*NONFAILING\((.*)\);\n`)
		out0 = re.ReplaceAllString(out0, "$1;\n")
	}
	if !opts.Debug {
		re := regexp.MustCompile(`\t*debug\(.*\);\n`)
		out0 = re.ReplaceAllString(out0, "")
		re = regexp.MustCompile(`\t*debug_dump_data\(.*\);\n`)
		out0 = re.ReplaceAllString(out0, "")
	}

	// Remove duplicate new lines.
	out1 := []byte(out0)
	for {
		out2 := bytes.Replace(out1, []byte{'\n', '\n', '\n'}, []byte{'\n', '\n'}, -1)
		if len(out1) == len(out2) {
			break
		}
		out1 = out2
	}

	return out1, nil
}

func generateTestFunc(w io.Writer, opts Options, calls []string, name string) {
	if !opts.Threaded && !opts.Collide {
		fmt.Fprintf(w, "void %v()\n{\n", name)
		if opts.Debug {
			// Use debug to avoid: error: ‘debug’ defined but not used.
			fmt.Fprintf(w, "\tdebug(\"%v\\n\");\n", name)
		}
		if opts.Repro {
			fmt.Fprintf(w, "\tsyscall(SYS_write, 1, \"executing program\\n\", strlen(\"executing program\\n\"));\n")
		}
		fmt.Fprintf(w, "\tmemset(r, -1, sizeof(r));\n")
		for _, c := range calls {
			fmt.Fprintf(w, "%s", c)
		}
		fmt.Fprintf(w, "}\n\n")
	} else {
		fmt.Fprintf(w, "void *thr(void *arg)\n{\n")
		fmt.Fprintf(w, "\tswitch ((long)arg) {\n")
		for i, c := range calls {
			fmt.Fprintf(w, "\tcase %v:\n", i)
			fmt.Fprintf(w, "%s", strings.Replace(c, "\t", "\t\t", -1))
			fmt.Fprintf(w, "\t\tbreak;\n")
		}
		fmt.Fprintf(w, "\t}\n")
		fmt.Fprintf(w, "\treturn 0;\n}\n\n")

		fmt.Fprintf(w, "void %v()\n{\n", name)
		fmt.Fprintf(w, "\tlong i;\n")
		fmt.Fprintf(w, "\tpthread_t th[%v];\n", 2*len(calls))
		fmt.Fprintf(w, "\n")
		if opts.Debug {
			// Use debug to avoid: error: ‘debug’ defined but not used.
			fmt.Fprintf(w, "\tdebug(\"%v\\n\");\n", name)
		}
		if opts.Repro {
			fmt.Fprintf(w, "\tsyscall(SYS_write, 1, \"executing program\\n\", strlen(\"executing program\\n\"));\n")
		}
		fmt.Fprintf(w, "\tmemset(r, -1, sizeof(r));\n")
		if opts.Collide {
			fmt.Fprintf(w, "\tsrand(getpid());\n")
		}
		fmt.Fprintf(w, "\tfor (i = 0; i < %v; i++) {\n", len(calls))
		fmt.Fprintf(w, "\t\tpthread_create(&th[i], 0, thr, (void*)i);\n")
		fmt.Fprintf(w, "\t\tusleep(rand()%%10000);\n")
		fmt.Fprintf(w, "\t}\n")
		if opts.Collide {
			fmt.Fprintf(w, "\tfor (i = 0; i < %v; i++) {\n", len(calls))
			fmt.Fprintf(w, "\t\tpthread_create(&th[%v+i], 0, thr, (void*)i);\n", len(calls))
			fmt.Fprintf(w, "\t\tif (rand()%%2)\n")
			fmt.Fprintf(w, "\t\t\tusleep(rand()%%10000);\n")
			fmt.Fprintf(w, "\t}\n")
		}
		fmt.Fprintf(w, "\tusleep(rand()%%100000);\n")
		fmt.Fprintf(w, "}\n\n")
	}
}

func generateCalls(exec []byte, opts Options) ([]string, int) {
	read := func() uint64 {
		if len(exec) < 8 {
			panic("exec program overflow")
		}
		v := *(*uint64)(unsafe.Pointer(&exec[0]))
		exec = exec[8:]
		return v
	}
	resultRef := func() string {
		arg := read()
		res := fmt.Sprintf("r[%v]", arg)
		if opDiv := read(); opDiv != 0 {
			res = fmt.Sprintf("%v/%v", res, opDiv)
		}
		if opAdd := read(); opAdd != 0 {
			res = fmt.Sprintf("%v+%v", res, opAdd)
		}
		return res
	}
	lastCall := 0
	seenCall := false
	var calls []string
	w := new(bytes.Buffer)
	newCall := func() {
		if seenCall {
			seenCall = false
			calls = append(calls, w.String())
			w = new(bytes.Buffer)
		}
	}
	n := 0
loop:
	for ; ; n++ {
		switch instr := read(); instr {
		case prog.ExecInstrEOF:
			break loop
		case prog.ExecInstrCopyin:
			newCall()
			addr := read()
			typ := read()
			size := read()
			switch typ {
			case prog.ExecArgConst:
				arg := read()
				bfOff := read()
				bfLen := read()
				if bfOff == 0 && bfLen == 0 {
					fmt.Fprintf(w, "\tNONFAILING(*(uint%v_t*)0x%x = (uint%v_t)0x%x);\n", size*8, addr, size*8, arg)
				} else {
					fmt.Fprintf(w, "\tNONFAILING(STORE_BY_BITMASK(uint%v_t, 0x%x, 0x%x, %v, %v));\n", size*8, addr, arg, bfOff, bfLen)
				}
			case prog.ExecArgResult:
				fmt.Fprintf(w, "\tNONFAILING(*(uint%v_t*)0x%x = %v);\n", size*8, addr, resultRef())
			case prog.ExecArgData:
				data := exec[:size]
				exec = exec[(size+7)/8*8:]
				var esc []byte
				for _, v := range data {
					hex := func(v byte) byte {
						if v < 10 {
							return '0' + v
						}
						return 'a' + v - 10
					}
					esc = append(esc, '\\', 'x', hex(v>>4), hex(v<<4>>4))
				}
				fmt.Fprintf(w, "\tNONFAILING(memcpy((void*)0x%x, \"%s\", %v));\n", addr, esc, size)
			case prog.ExecArgCsum:
				csum_kind := read()
				switch csum_kind {
				case prog.ExecArgCsumInet:
					fmt.Fprintf(w, "\tstruct csum_inet csum_%d;\n", n)
					fmt.Fprintf(w, "\tcsum_inet_init(&csum_%d);\n", n)
					csumChunksNum := read()
					for i := uint64(0); i < csumChunksNum; i++ {
						chunk_kind := read()
						chunk_value := read()
						chunk_size := read()
						switch chunk_kind {
						case prog.ExecArgCsumChunkData:
							fmt.Fprintf(w, "\tNONFAILING(csum_inet_update(&csum_%d, (const uint8_t*)0x%x, %d));\n", n, chunk_value, chunk_size)
						case prog.ExecArgCsumChunkConst:
							fmt.Fprintf(w, "\tuint%d_t csum_%d_chunk_%d = 0x%x;\n", chunk_size*8, n, i, chunk_value)
							fmt.Fprintf(w, "\tcsum_inet_update(&csum_%d, (const uint8_t*)&csum_%d_chunk_%d, %d);\n", n, n, i, chunk_size)
						default:
							panic(fmt.Sprintf("unknown checksum chunk kind %v", chunk_kind))
						}
					}
					fmt.Fprintf(w, "\tNONFAILING(*(uint16_t*)0x%x = csum_inet_digest(&csum_%d));\n", addr, n)
				default:
					panic(fmt.Sprintf("unknown csum kind %v", csum_kind))
				}
			default:
				panic(fmt.Sprintf("bad argument type %v", instr))
			}
		case prog.ExecInstrCopyout:
			addr := read()
			size := read()
			fmt.Fprintf(w, "\tif (r[%v] != -1)\n", lastCall)
			fmt.Fprintf(w, "\t\tNONFAILING(r[%v] = *(uint%v_t*)0x%x);\n", n, size*8, addr)
		default:
			// Normal syscall.
			newCall()
			if opts.Fault && opts.FaultCall == len(calls) {
				fmt.Fprintf(w, "\twrite_file(\"/sys/kernel/debug/failslab/ignore-gfp-wait\", \"N\");\n")
				fmt.Fprintf(w, "\twrite_file(\"/sys/kernel/debug/fail_futex/ignore-private\", \"N\");\n")
				fmt.Fprintf(w, "\tinject_fault(%v);\n", opts.FaultNth)
			}
			meta := sys.Calls[instr]
			emitCall := true
			if meta.CallName == "syz_test" {
				emitCall = false
			}
			if !opts.EnableTun && (meta.CallName == "syz_emit_ethernet" || meta.CallName == "syz_extract_tcp_res") {
				emitCall = false
			}
			native := !strings.HasPrefix(meta.CallName, "syz_")
			if emitCall {
				if native {
					fmt.Fprintf(w, "\tr[%v] = syscall(__NR_%v", n, meta.CallName)
				} else {
					fmt.Fprintf(w, "\tr[%v] = %v(", n, meta.CallName)
				}
			}
			nargs := read()
			for i := uint64(0); i < nargs; i++ {
				typ := read()
				size := read()
				_ = size
				if emitCall && (native || i > 0) {
					fmt.Fprintf(w, ", ")
				}
				switch typ {
				case prog.ExecArgConst:
					value := read()
					if emitCall {
						fmt.Fprintf(w, "0x%xul", value)
					}
					// Bitfields can't be args of a normal syscall, so just ignore them.
					read() // bit field offset
					read() // bit field length
				case prog.ExecArgResult:
					ref := resultRef()
					if emitCall {
						fmt.Fprintf(w, "%v", ref)
					}
				default:
					panic(fmt.Sprintf("unknown arg type %v", typ))
				}
			}
			if emitCall {
				fmt.Fprintf(w, ");\n")
			}
			lastCall = n
			seenCall = true
		}
	}
	newCall()
	return calls, n
}

func preprocessCommonHeader(opts Options, handled map[string]uint64, useBitmasks, useChecksums bool) (string, error) {
	var defines []string
	if useBitmasks {
		defines = append(defines, "SYZ_USE_BITMASKS")
	}
	if useChecksums {
		defines = append(defines, "SYZ_USE_CHECKSUMS")
	}
	switch opts.Sandbox {
	case "":
		// No sandbox, do nothing.
	case "none":
		defines = append(defines, "SYZ_SANDBOX_NONE")
	case "setuid":
		defines = append(defines, "SYZ_SANDBOX_SETUID")
	case "namespace":
		defines = append(defines, "SYZ_SANDBOX_NAMESPACE")
	default:
		return "", fmt.Errorf("unknown sandbox mode: %v", opts.Sandbox)
	}
	if opts.Threaded {
		defines = append(defines, "SYZ_THREADED")
	}
	if opts.Collide {
		defines = append(defines, "SYZ_COLLIDE")
	}
	if opts.Repeat {
		defines = append(defines, "SYZ_REPEAT")
	}
	if opts.Fault {
		defines = append(defines, "SYZ_FAULT_INJECTION")
	}
	if opts.EnableTun {
		defines = append(defines, "SYZ_TUN_ENABLE")
	}
	if opts.UseTmpDir {
		defines = append(defines, "SYZ_USE_TMP_DIR")
	}
	if opts.HandleSegv {
		defines = append(defines, "SYZ_HANDLE_SEGV")
	}
	if opts.WaitRepeat {
		defines = append(defines, "SYZ_WAIT_REPEAT")
	}
	if opts.Debug {
		defines = append(defines, "SYZ_DEBUG")
	}
	for name, _ := range handled {
		defines = append(defines, "__NR_"+name)
	}
	// TODO: need to know target arch + do cross-compilation
	defines = append(defines, "__x86_64__")

	cmd := exec.Command("cpp", "-nostdinc", "-undef", "-fdirectives-only", "-dDI", "-E", "-P", "-")
	for _, def := range defines {
		cmd.Args = append(cmd.Args, "-D"+def)
	}
	cmd.Stdin = strings.NewReader(commonHeader)
	stderr := new(bytes.Buffer)
	stdout := new(bytes.Buffer)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	if err := cmd.Run(); len(stdout.Bytes()) == 0 {
		return "", fmt.Errorf("cpp failed: %v\n%v\n%v\n", err, stdout.String(), stderr.String())
	}
	remove := append(defines, []string{
		"__STDC__",
		"__STDC_HOSTED__",
		"__STDC_UTF_16__",
		"__STDC_UTF_32__",
	}...)
	out := stdout.String()
	for _, def := range remove {
		out = strings.Replace(out, "#define "+def+" 1\n", "", -1)
	}
	// strip: #define __STDC_VERSION__ 201112L
	for _, def := range []string{"__STDC_VERSION__"} {
		pos := strings.Index(out, "#define "+def)
		if pos == -1 {
			continue
		}
		end := strings.IndexByte(out[pos:], '\n')
		if end == -1 {
			continue
		}
		out = strings.Replace(out, out[pos:end+1], "", -1)
	}
	return out, nil
}

// Build builds a C/C++ program from source src and returns name of the resulting binary.
// lang can be "c" or "c++".
func Build(lang, src string) (string, error) {
	bin, err := ioutil.TempFile("", "syzkaller")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	bin.Close()
	out, err := exec.Command("gcc", "-x", lang, "-Wall", "-Werror", src, "-o", bin.Name(), "-pthread", "-static", "-O1", "-g").CombinedOutput()
	if err != nil {
		// Some distributions don't have static libraries.
		out, err = exec.Command("gcc", "-x", lang, "-Wall", "-Werror", src, "-o", bin.Name(), "-pthread", "-O1", "-g").CombinedOutput()
	}
	if err != nil {
		os.Remove(bin.Name())
		data, _ := ioutil.ReadFile(src)
		return "", fmt.Errorf("failed to build program:\n%s\n%s", data, out)
	}
	return bin.Name(), nil
}

// Format reformats C source using clang-format.
func Format(src []byte) ([]byte, error) {
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	cmd := exec.Command("clang-format", "-assume-filename=/src.c", "-style", style)
	cmd.Stdin = bytes.NewReader(src)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return src, fmt.Errorf("failed to format source: %v\n%v", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// Something acceptable for kernel developers and email-friendly.
var style = `{
BasedOnStyle: LLVM,
IndentWidth: 2,
UseTab: Never,
BreakBeforeBraces: Linux,
IndentCaseLabels: false,
DerivePointerAlignment: false,
PointerAlignment: Left,
AlignTrailingComments: true,
AllowShortBlocksOnASingleLine: false,
AllowShortCaseLabelsOnASingleLine: false,
AllowShortFunctionsOnASingleLine: false,
AllowShortIfStatementsOnASingleLine: false,
AllowShortLoopsOnASingleLine: false,
ColumnLimit: 72,
}`
