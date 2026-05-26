#!/bin/bash
set -e

echo "Building CGO attestation wrapper library..."

INCLUDE_FLAGS="-I/usr/include/azguestattestation1 -I."

# Compile C++ sources
/usr/bin/g++ $INCLUDE_FLAGS -fPIC -c Logger.cpp -o Logger.o
/usr/bin/g++ $INCLUDE_FLAGS -fPIC -c attestation_wrapper.cpp -o attestation_wrapper.o

# Create static library
/usr/bin/ar rcs libattestation_wrapper.a Logger.o attestation_wrapper.o

echo "✓ Built libattestation_wrapper.a"

# Clean up object files
/bin/rm -f Logger.o attestation_wrapper.o
