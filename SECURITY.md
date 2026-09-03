# Security Policy

## Reporting a vulnerability

Please do not open a public issue for a security problem.

Report it privately through GitHub: open the repository's **Security** tab and
choose **Report a vulnerability**. You will get an acknowledgement within a week.
Fixes ship on `main` and in the next tagged release, and the advisory is
published once a fix is available.

## What counts

The parts of this repository that run on a learner's machine:

- `skills/tutor/scripts/tutor.py`: scaffolds curriculum files into a workspace
  and mutates state files there. Anything that lets a crafted curriculum or
  workspace write outside the workspace, follow symlinks it should not, or
  execute code during `init`, `sync`, or `status` is in scope.
- `check.sh` graders and exercise tests: these run the learner's own code by
  design. A grader that executes something other than the learner's exercise,
  or that reaches the network without saying so in the lesson, is in scope.
- `SKILL.md` and `TUTOR.md` files: prompt content that would make the tutor
  exfiltrate learner data or act outside the workspace is in scope.

## What does not count

Lessons deliberately contain insecure code, fake credentials, and vulnerable
patterns as teaching material, most visibly in the security lesson. Those are
not vulnerabilities in this project. Findings in the third-party tools the
curriculum teaches (Go, Docker, Kubernetes, Postgres) belong upstream.

## Supported versions

`main` and the latest tagged release.
