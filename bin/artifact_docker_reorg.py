#!/usr/bin/env python3
import argparse
import os
import re
import shutil
import sys
import tempfile
from pathlib import Path, PurePath


def unpack(archive: Path, base_output: Path) -> None:
    """
    Extracts an archive to the correct subdirectory under an output path.

    :param archive: Path of the archive to extract. Anything supported by
                    shutil.unpack_archive() will work.
    :param base_output: The root parent directory to extract to. Directories
                        will be created within this as necessary, e.g.
                        linux/arm/v6.
    """
    inner_dir = archive.name.removesuffix('.tar.gz').removesuffix('.zip') # expected to be xor
    _, platform = inner_dir.rsplit('.', 1)
    os, arch = platform.split('-', 1)
    final_dir = base_output / PurePath(os, arch)
    with tempfile.TemporaryDirectory() as tmp_dir:
        tmp_dir = Path(tmp_dir)
        shutil.unpack_archive(archive, tmp_dir)
        shutil.move(tmp_dir / inner_dir, final_dir)


def _parse_args(argv: [str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description='Untars archives to appropriate directories for Docker buildx.')
    parser.add_argument('input', help='Root path to read archives from')
    parser.add_argument('output', help='Root path to move binaries to')
    return parser.parse_args(argv[1:])


def main(argv: [str]) -> int:
    args = _parse_args(argv)
    output_dir = Path(args.output)
    for input_name in os.listdir(args.input):
        input_path = os.path.join(args.input, input_name)
        if not os.path.isfile(input_path):
            continue
        unpack(Path(input_path), output_dir)
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv))
