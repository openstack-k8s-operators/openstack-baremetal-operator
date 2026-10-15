"""Patch IPA standby.py to inject boot=UUID after bootc install for FIPS.

This script splices a small fixup method into
ironic_python_agent/extensions/standby.py and wires it into
execute_bootc_install so the BLS entry gets boot=UUID=<root-uuid> appended
whenever fips=1 is present without a boot= argument.
"""
import sys
import textwrap

# Dedent for readability, then re-indent as a class method. Without the
# final indent, textwrap.dedent leaves the method at column 0 and closes the
# StandbyExtension class early, nesting _download_container_and_bootc_install
# inside this helper.
FIXUP_METHOD = "\n" + textwrap.indent(textwrap.dedent("""\
    def _fixup_fips_boot_karg(self, device):
        \"\"\"Inject boot=UUID into BLS entry when fips=1 is set without boot=.

        Add boot=UUID when bootc leaves it out on FIPS installs.
        \"\"\"
        import tempfile as _tempfile

        try:
            output = utils.execute(
                'lsblk', '-nlo', 'PATH,FSTYPE,UUID', device,
                use_standard_locale=True)
        except Exception:
            LOG.warning('lsblk failed on %s, skipping FIPS boot karg fixup',
                        device)
            return

        root_part = root_uuid = None
        for line in output[0].strip().split('\\n'):
            fields = line.split()
            if len(fields) >= 3 and fields[1] in ('xfs', 'ext4', 'btrfs'):
                root_part, root_uuid = fields[0], fields[2]
                break

        if not root_part:
            return

        mnt = _tempfile.mkdtemp(prefix='ipa-fips-')
        try:
            utils.execute('mount', root_part, mnt)
            bls_dir = os.path.join(mnt, 'boot', 'loader', 'entries')
            if not os.path.isdir(bls_dir):
                return
            for name in os.listdir(bls_dir):
                if not name.endswith('.conf'):
                    continue
                path = os.path.join(bls_dir, name)
                with open(path) as fh:
                    text = fh.read()
                if 'fips=1' in text and 'boot=' not in text:
                    text = text.replace(
                        'fips=1', 'fips=1 boot=UUID=%s' % root_uuid)
                    with open(path, 'w') as fh:
                        fh.write(text)
                    LOG.info('Injected boot=UUID=%s into BLS %s for FIPS',
                             root_uuid, name)
        except Exception as exc:
            LOG.warning('FIPS boot karg fixup failed: %s', exc)
        finally:
            utils.execute('umount', mnt, check_exit_code=False)
            try:
                os.rmdir(mnt)
            except OSError:
                pass
"""), "    ") + "\n"

CALL_LINE = "        self._fixup_fips_boot_karg(device)\n"


def patch(filepath):
    with open(filepath) as fh:
        content = fh.read()

    # --- 1. Insert the method before _download_container_and_bootc_install ---
    marker = "    def _download_container_and_bootc_install("
    if marker not in content:
        print("ERROR: cannot find _download_container_and_bootc_install",
              file=sys.stderr)
        sys.exit(1)

    content = content.replace(marker, FIXUP_METHOD + marker, 1)

    # --- 2. Insert the call after _validate_partitioning in bootc path ---
    # We look for the pattern:
    #     _validate_partitioning(device)
    # followed within a few lines by the bootc-specific configdrive block.
    target = "        _validate_partitioning(device)\n"
    idx = content.find(target)
    while idx != -1:
        rest = content[idx + len(target):]
        # The bootc execute_bootc_install has 'Container image' nearby
        if "Container image" in rest[:600]:
            content = (content[:idx + len(target)]
                       + "\n" + CALL_LINE + "\n"
                       + content[idx + len(target):])
            break
        idx = content.find(target, idx + 1)
    else:
        print("ERROR: cannot find call-site in execute_bootc_install",
              file=sys.stderr)
        sys.exit(1)

    # Sanity-check: both helpers must remain class methods (4-space indent).
    if "    def _fixup_fips_boot_karg(self, device):" not in content:
        print("ERROR: fixup method missing class indentation", file=sys.stderr)
        sys.exit(1)
    if content.count(marker) != 1:
        print("ERROR: _download_container_and_bootc_install marker broken",
              file=sys.stderr)
        sys.exit(1)
    try:
        compile(content, filepath, "exec")
    except SyntaxError as exc:
        print("ERROR: patched file has syntax error: %s" % exc, file=sys.stderr)
        sys.exit(1)

    with open(filepath, "w") as fh:
        fh.write(content)
    print("Patched %s" % filepath)


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: %s <path/to/standby.py>" % sys.argv[0], file=sys.stderr)
        sys.exit(1)
    patch(sys.argv[1])
