package embedded

import (
	"fmt"
	"os"
	"path/filepath"
)

// runtime.go provides the firmware "runtime" — the linker script, Cortex-M
// startup code, and semihosting I/O — that a bare `arm-none-eabi-gcc main.c`
// does NOT produce. Without these, a compiled ELF has no valid vector table or
// reset handler, so QEMU faults on the first instruction and emulation prints
// nothing. The build tool generates them into <project>/.hcai_runtime/ and
// links them in automatically.
//
// Output is via ARM semihosting (the BKPT 0xAB trap), which QEMU implements at
// the CPU level — it works on every Cortex-M machine regardless of whether
// that machine models a UART. The model writes logs with the LOG() macro from
// hcai.h and may end emulation deterministically with hcai_exit().

// RuntimeProfile is the memory layout + QEMU machine for one chip family.
type RuntimeProfile struct {
	// FlashOrigin/FlashLength and RAMOrigin/RAMLength describe the chip's
	// address map; they go straight into the generated linker script.
	FlashOrigin string
	FlashLength string
	RAMOrigin   string
	RAMLength   string
	// QEMUMachine is the -machine value for `emulate`. Every value here is a
	// machine present in upstream/xPack QEMU 9.x — verified, not guessed.
	QEMUMachine string
	// CPUName is for human-readable build/emulate messages.
	CPUName string
}

// runtimeProfiles maps an arch string (the same keys build/emulate accept) to
// a verified memory layout + QEMU machine.
//
// STM32 families are mapped to the real STM32 machine whose core matches:
//   - F1/F3/F4/L4/G4/WB → olimex-stm32-h405 (STM32F405, Cortex-M4F) — the
//     best-supported STM32 machine in QEMU and the one verified to print.
//   - F0/G0/L0          → stm32vldiscovery  (STM32F100, Cortex-M3)
//
// Generic Cortex-M arches use ARM's MPS2 reference machines, which have solid
// QEMU support across M3/M4/M7/M33.
var runtimeProfiles = map[string]RuntimeProfile{
	// ── STM32 Cortex-M4 family → STM32F405 board ────────────────────────────
	"stm32f1": {"0x08000000", "1024K", "0x20000000", "128K", "olimex-stm32-h405", "STM32F405 (Cortex-M4F)"},
	"stm32f3": {"0x08000000", "1024K", "0x20000000", "128K", "olimex-stm32-h405", "STM32F405 (Cortex-M4F)"},
	"stm32f4": {"0x08000000", "1024K", "0x20000000", "128K", "olimex-stm32-h405", "STM32F405 (Cortex-M4F)"},
	"stm32l4": {"0x08000000", "1024K", "0x20000000", "128K", "olimex-stm32-h405", "STM32F405 (Cortex-M4F)"},
	"stm32g4": {"0x08000000", "1024K", "0x20000000", "128K", "olimex-stm32-h405", "STM32F405 (Cortex-M4F)"},
	"stm32wb": {"0x08000000", "1024K", "0x20000000", "128K", "olimex-stm32-h405", "STM32F405 (Cortex-M4F)"},
	// ── STM32 Cortex-M3 family → STM32F100 board ────────────────────────────
	"stm32f0": {"0x08000000", "128K", "0x20000000", "8K", "stm32vldiscovery", "STM32F100 (Cortex-M3)"},
	"stm32g0": {"0x08000000", "128K", "0x20000000", "8K", "stm32vldiscovery", "STM32F100 (Cortex-M3)"},
	"stm32l0": {"0x08000000", "128K", "0x20000000", "8K", "stm32vldiscovery", "STM32F100 (Cortex-M3)"},
	"stm32l1": {"0x08000000", "128K", "0x20000000", "8K", "stm32vldiscovery", "STM32F100 (Cortex-M3)"},
	// ── Generic Cortex-M → ARM MPS2 reference machines ──────────────────────
	"cortex-m3":  {"0x00000000", "4096K", "0x21000000", "4096K", "mps2-an385", "Cortex-M3 (MPS2)"},
	"cortex-m4":  {"0x00000000", "4096K", "0x21000000", "4096K", "mps2-an386", "Cortex-M4 (MPS2)"},
	"cortex-m4f": {"0x00000000", "4096K", "0x21000000", "4096K", "mps2-an386", "Cortex-M4F (MPS2)"},
	"cortex-m7":  {"0x00000000", "8192K", "0x21000000", "8192K", "mps2-an500", "Cortex-M7 (MPS2)"},
	"cortex-m7f": {"0x00000000", "8192K", "0x21000000", "8192K", "mps2-an500", "Cortex-M7 (MPS2)"},
	"cortex-m33": {"0x10000000", "4096K", "0x38000000", "4096K", "mps2-an505", "Cortex-M33 (MPS2)"},
}

// RuntimeProfileFor returns the runtime profile for an arch, if one exists.
func RuntimeProfileFor(arch string) (RuntimeProfile, bool) {
	p, ok := runtimeProfiles[arch]
	return p, ok
}

// runtimeDirName is the per-project folder holding the generated runtime.
const runtimeDirName = ".hcai_runtime"

// GenerateRuntime writes the linker script, startup code, and semihosting
// header/source for the given profile into <root>/.hcai_runtime/. It returns
// the directory, the list of extra .c files to compile, and the linker script
// path. Files are rewritten every build so a profile change always takes hold.
func GenerateRuntime(root string, p RuntimeProfile) (dir string, extraSources []string, linkerScript string, err error) {
	dir = filepath.Join(root, runtimeDirName)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, "", err
	}

	files := map[string]string{
		"hcai_linker.ld":  linkerScriptTemplate(p),
		"hcai_startup.c":  startupSource,
		"hcai_semihost.c": semihostSource,
		"hcai.h":          hcaiHeader,
	}
	for name, content := range files {
		if werr := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); werr != nil {
			return "", nil, "", werr
		}
	}

	linkerScript = filepath.Join(dir, "hcai_linker.ld")
	extraSources = []string{
		filepath.Join(dir, "hcai_startup.c"),
		filepath.Join(dir, "hcai_semihost.c"),
	}
	return dir, extraSources, linkerScript, nil
}

// linkerScriptTemplate builds a linker script for the profile's memory map.
// It places the vector table first in flash, then code/rodata, then a .data
// section loaded from flash but addressed in RAM, then zero-init .bss.
func linkerScriptTemplate(p RuntimeProfile) string {
	return fmt.Sprintf(`/* Auto-generated by hardcore-ai — do not edit; rewritten every build. */
ENTRY(Reset_Handler)
MEMORY {
  FLASH (rx) : ORIGIN = %s, LENGTH = %s
  RAM  (rwx) : ORIGIN = %s, LENGTH = %s
}
_estack = ORIGIN(RAM) + LENGTH(RAM);
SECTIONS {
  .isr_vector : { KEEP(*(.isr_vector)) } > FLASH
  .text : {
    *(.text*)
    *(.rodata*)
  } > FLASH
  _sidata = LOADADDR(.data);
  .data : {
    . = ALIGN(4);
    _sdata = .;
    *(.data*)
    . = ALIGN(4);
    _edata = .;
  } > RAM AT > FLASH
  .bss : {
    . = ALIGN(4);
    _sbss = .;
    *(.bss*)
    *(COMMON)
    . = ALIGN(4);
    _ebss = .;
  } > RAM
  . = ALIGN(8);
  _end = .;
}
`, p.FlashOrigin, p.FlashLength, p.RAMOrigin, p.RAMLength)
}

// startupSource is the Cortex-M reset path: a 16-entry vector table (initial
// SP + reset + system exceptions), and a Reset_Handler that initializes .data
// and .bss before calling main(). After main() returns it calls hcai_exit() so
// emulation stops cleanly instead of hanging until the timeout.
const startupSource = `/* Auto-generated by hardcore-ai — Cortex-M startup. Do not edit. */
#include <stdint.h>

extern uint32_t _sidata, _sdata, _edata, _sbss, _ebss, _estack;
extern int main(void);
void hcai_exit(int code);
void Reset_Handler(void);
void Default_Handler(void) { while (1) { } }

__attribute__((section(".isr_vector"), used))
void (*const hcai_vector_table[16])(void) = {
    (void (*)(void))(&_estack), /* 0: initial stack pointer */
    Reset_Handler,              /* 1: reset */
    Default_Handler,            /* 2: NMI */
    Default_Handler,            /* 3: HardFault */
    Default_Handler,            /* 4: MemManage */
    Default_Handler,            /* 5: BusFault */
    Default_Handler,            /* 6: UsageFault */
    0, 0, 0, 0,                 /* 7-10: reserved */
    Default_Handler,            /* 11: SVCall */
    Default_Handler,            /* 12: DebugMon */
    0,                          /* 13: reserved */
    Default_Handler,            /* 14: PendSV */
    Default_Handler,            /* 15: SysTick */
};

void Reset_Handler(void) {
    uint32_t *src = &_sidata;
    uint32_t *dst = &_sdata;
    while (dst < &_edata) { *dst++ = *src++; }
    for (dst = &_sbss; dst < &_ebss;) { *dst++ = 0; }
    int code = main();
    hcai_exit(code);
    while (1) { }
}
`

// semihostSource implements ARM semihosting calls used by the runtime: string
// output (SYS_WRITE0) and program exit (SYS_EXIT). printf is intentionally not
// pulled in — it would need a heap and a much larger libc; LOG() in hcai.h is
// a lightweight formatter built on these primitives.
const semihostSource = `/* Auto-generated by hardcore-ai — ARM semihosting I/O. Do not edit. */
#include <stdint.h>

static inline int hcai_sh_call(int op, void *arg) {
    register int r0 __asm__("r0") = op;
    register void *r1 __asm__("r1") = arg;
    __asm__ volatile("bkpt 0xAB" : "+r"(r0) : "r"(r1) : "memory");
    return r0;
}

/* SYS_WRITE0 (0x04): write a NUL-terminated string to the host console. */
void hcai_write0(const char *s) {
    hcai_sh_call(0x04, (void *)s);
}

/* SYS_EXIT (0x18): stop emulation. ADP_Stopped_ApplicationExit = 0x20026. */
void hcai_exit(int code) {
    uint32_t block[2] = { 0x20026, (uint32_t)code };
    hcai_sh_call(0x18, block);
    while (1) { }
}
`

// hcaiHeader is what firmware #includes. It exposes hcai_write0/hcai_exit and a
// printf-style LOG(fmt, ...) built on semihosting with no libc or heap. The
// model is told (in the system prompt) to #include "hcai.h" and use LOG().
const hcaiHeader = `/* Auto-generated by hardcore-ai. #include "hcai.h" for logging on emulation.
 * Output goes to the host via ARM semihosting and appears in emulate output.
 */
#ifndef HCAI_H
#define HCAI_H

#include <stdarg.h>

void hcai_write0(const char *s);
void hcai_exit(int code);

/* hcai_vlog: a tiny freestanding printf. Supports %s %d %u %x %c %%.
 * No heap, no libc. Output is flushed as one semihosting write. */
static inline void hcai_vlog(const char *fmt, va_list ap) {
    char buf[256];
    int n = 0;
    for (const char *p = fmt; *p && n < (int)sizeof(buf) - 1; p++) {
        if (*p != '%') { buf[n++] = *p; continue; }
        p++;
        if (*p == 0) break;
        if (*p == '%') { buf[n++] = '%'; continue; }
        if (*p == 'c') { buf[n++] = (char)va_arg(ap, int); continue; }
        if (*p == 's') {
            const char *s = va_arg(ap, const char *);
            if (!s) s = "(null)";
            while (*s && n < (int)sizeof(buf) - 1) buf[n++] = *s++;
            continue;
        }
        /* numeric: %d %u %x */
        unsigned long v;
        int base = 10, neg = 0;
        if (*p == 'd') {
            long sv = va_arg(ap, int);
            if (sv < 0) { neg = 1; v = (unsigned long)(-sv); }
            else v = (unsigned long)sv;
        } else if (*p == 'u') {
            v = (unsigned long)va_arg(ap, unsigned int);
        } else if (*p == 'x') {
            v = (unsigned long)va_arg(ap, unsigned int);
            base = 16;
        } else {
            buf[n++] = *p; /* unknown specifier — emit literally */
            continue;
        }
        char tmp[24];
        int t = 0;
        if (v == 0) tmp[t++] = '0';
        while (v > 0) {
            int d = (int)(v % (unsigned)base);
            tmp[t++] = (char)(d < 10 ? '0' + d : 'a' + d - 10);
            v /= (unsigned)base;
        }
        if (neg && n < (int)sizeof(buf) - 1) buf[n++] = '-';
        while (t > 0 && n < (int)sizeof(buf) - 1) buf[n++] = tmp[--t];
    }
    buf[n] = 0;
    hcai_write0(buf);
}

/* LOG(fmt, ...) — printf-style logging visible in emulation. */
static inline void LOG(const char *fmt, ...) {
    va_list ap;
    va_start(ap, fmt);
    hcai_vlog(fmt, ap);
    va_end(ap);
}

#endif /* HCAI_H */
`
