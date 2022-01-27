// Copyright 2018 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package linux_test

import (
	"testing"

	"github.com/google/syzkaller/prog"
	_ "github.com/google/syzkaller/sys/linux/gen"
	"github.com/google/syzkaller/sys/targets"
)

func TestNeutralize(t *testing.T) {
	prog.TestDeserializeHelper(t, targets.Linux, targets.AMD64, nil, []prog.DeserializeTest{
		{
			In:  `syslog(0x10000000006, 0x0, 0x0)`,
			Out: `syslog(0x9, 0x0, 0x0)`,
		},
		{
			In:  `syslog(0x10000000007, 0x0, 0x0)`,
			Out: `syslog(0x9, 0x0, 0x0)`,
		},
		{
			In: `syslog(0x1, 0x0, 0x0)`,
		},

		{
			In:  `ptrace(0xf000000000, 0x0)`,
			Out: `ptrace(0xffffffffffffffff, 0x0)`,
		},
		{
			In:  `ptrace$peek(0x0, 0x0, &(0x7f0000000000))`,
			Out: `ptrace$peek(0xffffffffffffffff, 0x0, &(0x7f0000000000))`,
		},
		{
			In: `ptrace(0x1, 0x0)`,
		},
		{
			In:  `arch_prctl$ARCH_SET_GS(0xf00000001002, 0x0)`,
			Out: `arch_prctl$ARCH_SET_GS(0x1001, 0x0)`,
		},
		{
			In: `arch_prctl$ARCH_SET_GS(0x1003, 0x0)`,
		},
		{
			In:  `ioctl(0x0, 0x200000c0045877, 0x0)`,
			Out: `ioctl(0x0, 0xc0045878, 0x0)`,
		},
		{
			In:  `ioctl$int_in(0x0, 0x2000008004587d, 0x0)`,
			Out: `ioctl$int_in(0x0, 0x6609, 0x0)`,
		},
		{
			In:  `fanotify_mark(0x1, 0x2, 0x407fe029, 0x3, 0x0)`,
			Out: `fanotify_mark(0x1, 0x2, 0x4078e029, 0x3, 0x0)`,
		},
		{
			In: `fanotify_mark(0xffffffffffffffff, 0xffffffffffffffff, 0xfffffffffff8ffff, 0xffffffffffffffff, 0x0)`,
		},
		{
			In:  `syz_init_net_socket$bt_hci(0x1, 0x0, 0x0)`,
			Out: `syz_init_net_socket$bt_hci(0xffffffffffffffff, 0x0, 0x0)`,
		},
		{
			In: `syz_init_net_socket$bt_hci(0x27, 0x0, 0x0)`,
		},
		{
			In: `syz_init_net_socket$bt_hci(0x1a, 0x0, 0x0)`,
		},
		{
			In: `syz_init_net_socket$bt_hci(0x1f, 0x0, 0x0)`,
		},
		{
			In:  `mmap(0x0, 0x0, 0x0, 0x0, 0x0, 0x0)`,
			Out: `mmap(0x0, 0x0, 0x0, 0x10, 0x0, 0x0)`,
		},
		{
			In: `mremap(0x0, 0x0, 0x0, 0xcc, 0x0)`,
		},
		{
			In:  `mremap(0x0, 0x0, 0x0, 0xcd, 0x0)`,
			Out: `mremap(0x0, 0x0, 0x0, 0xcf, 0x0)`,
		},
		{
			In: `
mknod(0x0, 0x1000, 0x0)
mknod(0x0, 0x8000, 0x0)
mknod(0x0, 0xc000, 0x0)
mknod(0x0, 0x2000, 0x0)
mknod(0x0, 0x6000, 0x0)
mknod(0x0, 0x6000, 0x700)
`,
			Out: `
mknod(0x0, 0x1000, 0x0)
mknod(0x0, 0x8000, 0x0)
mknod(0x0, 0xc000, 0x0)
mknod(0x0, 0x8000, 0x0)
mknod(0x0, 0x8000, 0x0)
mknod(0x0, 0x6000, 0x700)
`,
		},
		{
			In: `
exit(0x3)
exit(0x43)
exit(0xc3)
exit(0xc3)
exit_group(0x5a)
exit_group(0x43)
exit_group(0x443)
`,
			Out: `
exit(0x3)
exit(0x1)
exit(0x1)
exit(0x1)
exit_group(0x5a)
exit_group(0x1)
exit_group(0x1)
`,
		},
		{
			In: `
syz_open_dev$tty1(0xc, 0x4, 0x4)
syz_open_dev$tty1(0xb, 0x2, 0x4)
syz_open_dev$tty1(0xc, 0x4, 0x5)
`,
			Out: `
syz_open_dev$tty1(0xc, 0x4, 0x4)
syz_open_dev$tty1(0xc, 0x4, 0x4)
syz_open_dev$tty1(0xc, 0x4, 0x1)
`,
		},
		{
			In: `syz_open_dev$MSR(0x0, 0x0, 0x0)`,
		},
		{
			In: `
ioctl$X86_IOC_RDMSR_REGS(0xa, 0xc02063a0, 0x0)
ioctl$X86_IOC_RDMSR_REGS(0xa, 0xc02063a1, 0x0)
`,
			Out: `
ioctl$X86_IOC_RDMSR_REGS(0xa, 0xc02063a0, 0x0)
ioctl$X86_IOC_RDMSR_REGS(0xa, 0xc02063a0, 0x0)
`,
		},
		{
			In:  `sched_setattr(0x0, &(0x7f00000002c0)={0x0, 0x1, 0x0, 0x0, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0}, 0x0)`,
			Out: `sched_setattr(0x0, &(0x7f00000002c0)={0x0, 0x0, 0x0, 0x0, 0x3}, 0x0)`,
		},
		{
			In:  `sched_setattr(0x0, &(0x7f00000002c0)={0x0, 0x2, 0x0, 0x0, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0}, 0x0)`,
			Out: `sched_setattr(0x0, &(0x7f00000002c0)={0x0, 0x0, 0x0, 0x0, 0x3}, 0x0)`,
		},
		{
			In:  `sched_setattr(0x0, &(0x7f00000002c0)={0x0, 0x3, 0x0, 0x0, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0}, 0x0)`,
			Out: `sched_setattr(0x0, &(0x7f00000002c0)={0x0, 0x3, 0x0, 0x0, 0x3}, 0x0)`,
		},
		{
			In:  `sched_setattr(0x0, 0x123456, 0x0)`,
			Out: `sched_setattr(0x0, 0x0, 0x0)`,
		},
		{
			In:  `sched_setattr(0x0, &(0x7f00000001c0)=ANY=[@ANYBLOB="1234567812345678"], 0x0)`,
			Out: `sched_setattr(0x0, &(0x7f00000001c0)=ANY=[@ANYBLOB='\x00\x00\x00\x00\x00\x00\x00\x00'], 0x0)`,
		},
	})
}
