#!/usr/bin/env python3

# 개인적으로 자주 사용하던 파일입니다

import argparse
import subprocess as sp

parser = argparse.ArgumentParser(description="Runs docker compose. usage: ./docker-compose.py up")
parser.add_argument(
    "-e",
    type=str,
    help="environment",
    default="local",
    choices=["local"]
)
parser.add_argument("-v", help="verbose output", action="store_true", default=True)

args, unknown = parser.parse_known_args()
if args.v:
    print(args, unknown)

project_name = f"coupons_{args.e}"
yaml_file = f"./docker/docker-compose.yaml"

commands = ["DOCKER_BUILDKIT=1", "docker", "compose", "-f", yaml_file, "-p", project_name]
commands.extend(unknown)

process = sp.run(" ".join(commands), shell=True)

exit(process.returncode)

