"""Real child pipes must not depend on the differential caller's locale.

Only executable selection is adapted: build/program output comes from actual
Python child processes. Missing text encodings simulate a legacy CP936 host.
No Hybrid compiler, Go toolchain, network or external credentials are used.
"""
import contextlib
import importlib.util
import io
from pathlib import Path
import subprocess
import sys
import tempfile
import threading
import unittest
from unittest.mock import patch


HARNESS = Path(__file__).with_name("differential_loop.py")
SPEC = importlib.util.spec_from_file_location("diff_encoding_harness", HARNESS)
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)
REAL_RUN = subprocess.run


class RealProcessEncodingTests(unittest.TestCase):
    def run_corpus(self, *, build_stderr=b"", build_exit=0,
                   program_stdout=b"ok\n", expected_output="ok"):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            corpus = root / "corpus"
            corpus.mkdir()
            marker = ("" if expected_output is None else
                      f"// EXPECTED-OUTPUT: {expected_output}\n")
            (corpus / "00-first.hy").write_text(
                "// EXPECTED-VERDICT: ACCEPT\n" + marker + "fn main() {}\n",
                encoding="utf-8",
            )
            (corpus / "01-control.hy").write_text(
                "// EXPECTED-VERDICT: ACCEPT\n// EXPECTED-OUTPUT: ok\nfn main() {}\n",
                encoding="utf-8",
            )
            compiler = root / "hyb"
            compiler.touch()
            active_first = False

            def launch(command, **kwargs):
                nonlocal active_first
                if len(command) > 1 and command[1] == "build":
                    active_first = Path(command[2]).name == "00-first.hy"
                    (Path(kwargs["cwd"]) / "a.out").touch()
                    payload = build_stderr if active_first else b""
                    code = build_exit if active_first else 0
                    child = ("import sys; sys.stderr.buffer.write(" + repr(payload)
                             + "); sys.exit(" + str(code) + ")")
                else:
                    payload = program_stdout if active_first else b"ok\n"
                    child = "import sys; sys.stdout.buffer.write(" + repr(payload) + ")"
                if kwargs.get("text") and not kwargs.get("encoding"):
                    kwargs["encoding"] = "cp936"
                return REAL_RUN([sys.executable, "-c", child], **kwargs)

            output = io.StringIO()
            pipe_errors = []
            with patch.object(sys, "argv", [str(HARNESS), "--bin", str(compiler),
                                           "--corpus", str(corpus)]), \
                    patch.object(MODULE.subprocess, "run", side_effect=launch), \
                    patch.object(threading, "excepthook", side_effect=pipe_errors.append), \
                    contextlib.redirect_stdout(output):
                result = MODULE.main()
            self.assertEqual(pipe_errors, [], "subprocess pipe reader raised an exception")
            return result, output.getvalue()

    def test_utf8_program_output_survives_a_legacy_host_locale(self):
        result, output = self.run_corpus(program_stdout="错\n".encode("utf-8"),
                                         expected_output="错")
        self.assertEqual(result, 0, output)
        self.assertIn("build+run): 2/2", output)

    def test_utf8_build_diagnostics_do_not_abort_a_successful_case(self):
        result, output = self.run_corpus(build_stderr="错\n".encode("utf-8"))
        self.assertEqual(result, 0, output)
        self.assertIn("FULL AGREEMENT", output)

    def test_invalid_utf8_cannot_match_a_registered_replacement_character(self):
        result, output = self.run_corpus(program_stdout=b"\xff\n", expected_output="\ufffd")
        self.assertEqual(result, 1, output)
        self.assertIn("not valid UTF-8", output)
        self.assertIn("PASS  01-control.hy", output)
        self.assertIn("build+run): 1/2", output)

    def test_invalid_build_diagnostic_still_reports_failure_and_next_case(self):
        result, output = self.run_corpus(build_stderr=b"\xff", build_exit=1)
        self.assertEqual(result, 1, output)
        self.assertIn("build failed", output)
        self.assertIn("PASS  01-control.hy", output)
        self.assertIn("build+run): 1/2", output)

    def test_binary_stdout_without_an_output_assertion_is_not_text_decoded(self):
        result, output = self.run_corpus(program_stdout=b"\xff", expected_output=None)
        self.assertEqual(result, 0, output)
        self.assertIn("build+run): 2/2", output)

    def test_ascii_control_keeps_full_agreement(self):
        result, output = self.run_corpus()
        self.assertEqual(result, 0, output)
        self.assertIn("FULL AGREEMENT", output)


if __name__ == "__main__":
    unittest.main()
