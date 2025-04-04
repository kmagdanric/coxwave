#!/bin/bash

# remove old generated files
rm -rf proto/couponpbconnect
rm -rf proto/coupon.pb.go

# Generate code using buf
buf generate

if [ $? -eq 0 ]; then
    echo "Protocol buffer code generated successfully!"
else
    echo "Error: Protocol buffer code generation failed"
    exit 1
fi
