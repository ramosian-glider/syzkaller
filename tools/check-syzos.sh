#!/bin/sh
#
# This script scans the syz-executor binary for data relocations accesses
# within the "guest" ELF section that are problematic for the SYZOS guest 
# code.
#
# It uses $TARGETOS and $TARGETARCH to locate the binary and determine the
# correct architecture.
#

set -e

SECTION_TO_CHECK="guest"

echoerr() {
    echo "$@" >&2
}

if [ "$TARGETOS" != "linux" ]; then
    echo "[INFO] TARGETOS is '$TARGETOS', not 'linux'. Skipping check."
    exit 0
fi

if [ -z "$TARGETARCH" ]; then
    echoerr "Error: \$TARGETARCH environment variable is not set."
    exit 1
fi

BINARY="bin/${TARGETOS}_${TARGETARCH}/syz-executor"

if [ ! -f "$BINARY" ]; then
    echoerr "Error: Binary not found at '$BINARY'"
    exit 1
fi

echoerr "--> Analyzing architecture '$TARGETARCH'..."
OBJDUMP_CMD=""

if [ "$TARGETARCH" = "amd64" ]; then
    ARCH="x86_64"
    PATTERNS_TO_FIND='\(%rip\)'
    if command -v x86_64-linux-gnu-objdump > /dev/null; then
        OBJDUMP_CMD="x86_64-linux-gnu-objdump"
    fi
elif [ "$TARGETARCH" = "arm64" ]; then
    ARCH="aarch64"
    PATTERNS_TO_FIND='adrp'
    if command -v aarch64-linux-gnu-objdump > /dev/null; then
        OBJDUMP_CMD="aarch64-linux-gnu-objdump"
    fi
else
    echo "[INFO] Unsupported architecture '$TARGETARCH', skipping check."
    exit 0
fi
echoerr "--> Detected architecture: $ARCH"

if [ -z "$OBJDUMP_CMD" ]; then
    echoerr "--> Arch-specific objdump not found, falling back to generic 'objdump'..."
    if command -v objdump > /dev/null; then
        OBJDUMP_CMD="objdump"
    fi
fi

if [ -z "$OBJDUMP_CMD" ]; then
    echoerr "Error: Could not find a usable objdump binary."
    exit 1
fi
echoerr "--> Using objdump: $OBJDUMP_CMD"

echoerr "--> Verifying existence of section '$SECTION_TO_CHECK' in '$BINARY'..."
if ! "$OBJDUMP_CMD" -h --section="$SECTION_TO_CHECK" "$BINARY" >/dev/null 2>&1; then
    echo
    echo "[INFO] Section '$SECTION_TO_CHECK' not found in '$BINARY'. Skipping check."
    exit 0
fi

echoerr "--> Disassembling section '$SECTION_TO_CHECK' and scanning for patterns ('$PATTERNS_TO_FIND')..."

DISASSEMBLY_STATUS=0
DISASSEMBLY_OUTPUT=$("$OBJDUMP_CMD" -d --section="$SECTION_TO_CHECK" "$BINARY" 2>/dev/null) || DISASSEMBLY_STATUS=$?

if [ $DISASSEMBLY_STATUS -ne 0 ]; then
    echoerr "Error: '$OBJDUMP_CMD' failed to disassemble the '$SECTION_TO_CHECK' section."
    # Attempt to show the actual error to the user
    "$OBJDUMP_CMD" -d --section="$SECTION_TO_CHECK" "$BINARY" >/dev/null
    exit 1
fi

FOUND_INSTRUCTIONS=$(echo "$DISASSEMBLY_OUTPUT" | grep -E "$PATTERNS_TO_FIND" || true)

if [ -n "$FOUND_INSTRUCTIONS" ]; then
    echo
    echo "------------------------------------------------------------------"
    echo "[FAIL] Found problematic data access instructions in '$SECTION_TO_CHECK'."
    echo "The following instructions are likely to cause crashes in SyzOS:"
    echo "$FOUND_INSTRUCTIONS" | sed 's/^/  /'
    echo "------------------------------------------------------------------"
    echo
    echo "This typically happens when the C compiler emits read-only constants for"
    echo "zero-initializing structs or for jump tables in switch statements."
    exit 1
else
    echo
    echo "[OK] No problematic data access instructions found in '$SECTION_TO_CHECK'."
    exit 0
fi
