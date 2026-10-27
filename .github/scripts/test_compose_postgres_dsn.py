import subprocess
import sys
import unittest
from pathlib import Path


SCRIPT = Path(__file__).with_name("compose-postgres-dsn.py")


class ComposePostgresDSNTest(unittest.TestCase):
    def run_helper(self, contents: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(SCRIPT)],
            input=contents,
            capture_output=True,
            text=True,
            check=False,
        )

    def test_composes_postgres_dsn_and_encodes_password(self) -> None:
        result = self.run_helper(
            "POSTGRES_HOST=127.0.0.1\n"
            "POSTGRES_PORT=5432\n"
            "POSTGRES_USER=postgres\n"
            "POSTGRES_DB=usgate\n"
            "POSTGRES_PASSWORD=p@ss/word\n"
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn(
            "SQL_DSN=postgresql://postgres:p%40ss%2Fword@127.0.0.1:5432/usgate",
            result.stdout,
        )
        dsn_line = next(line for line in result.stdout.splitlines() if line.startswith("SQL_DSN="))
        self.assertNotIn("p@ss/word", dsn_line)

    def test_preserves_explicit_sql_dsn(self) -> None:
        contents = "SQL_DSN=postgresql://db.example/usgate\nPOSTGRES_HOST=other\n"

        result = self.run_helper(contents)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, contents)

    def test_replaces_empty_sql_dsn(self) -> None:
        result = self.run_helper(
            "SQL_DSN=\n"
            "POSTGRES_HOST=127.0.0.1\n"
            "POSTGRES_PORT=5432\n"
            "POSTGRES_USER=postgres\n"
            "POSTGRES_DB=usgate\n"
            "POSTGRES_PASSWORD=secret\n"
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.count("SQL_DSN="), 1)
        self.assertIn("SQL_DSN=postgresql://", result.stdout)

    def test_rejects_partial_postgres_settings_without_echoing_values(self) -> None:
        result = self.run_helper("POSTGRES_HOST=127.0.0.1\nPOSTGRES_PASSWORD=secret\n")

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("incomplete PostgreSQL settings", result.stderr)
        self.assertNotIn("secret", result.stderr)

    def test_leaves_non_postgres_environment_unchanged(self) -> None:
        contents = "SESSION_SECRET=placeholder\n"

        result = self.run_helper(contents)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, contents)


if __name__ == "__main__":
    unittest.main()
