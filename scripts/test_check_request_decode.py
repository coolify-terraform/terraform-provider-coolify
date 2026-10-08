#!/usr/bin/env python3
"""Unit tests for scripts/check-request-decode.py."""

import importlib.util
import unittest
from pathlib import Path

_SCRIPT = Path(__file__).resolve().parent / "check-request-decode.py"
_spec = importlib.util.spec_from_file_location("check_request_decode", _SCRIPT)
rd = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(rd)

OMITTED_INT = '''
func handler(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Timeout int64 `json:"timeout,omitempty"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        return
    }
}
'''

POINTER_INT = '''
func handler(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Timeout *int64 `json:"timeout,omitempty"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        return
    }
}
'''


class TestRequestDecode(unittest.TestCase):
    def test_omitempty_value_is_rejected(self):
        self.assertTrue(rd.problems_in_source(OMITTED_INT, "example_test.go"))

    def test_pointer_is_allowed(self):
        self.assertEqual(rd.problems_in_source(POINTER_INT, "example_test.go"), [])

    def test_decode_optional_is_allowed(self):
        src = '''
func handler(w http.ResponseWriter, r *http.Request) {
    raw, _ := io.ReadAll(r.Body)
    obj, _ := acctest.DecodeObject(bytes.NewReader(raw))
    timeout, _ := acctest.DecodeOptional[int64](obj, "timeout", nil)
    _ = timeout
}
'''
        self.assertEqual(rd.problems_in_source(src, "example_test.go"), [])

    def test_current_tree_passes(self):
        self.assertEqual(rd.check_repo(), [])


if __name__ == "__main__":
    unittest.main()
