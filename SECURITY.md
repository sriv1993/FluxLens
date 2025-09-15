# Security Policy

## Supported Versions

FluxLens is in early active development. The `main` branch is the only
supported version. Tagged releases will be supported once Phase 2 GA
(v1.0.0) ships.

## Reporting a Vulnerability

If you discover a security vulnerability in FluxLens, please report it
privately so we can address it before public disclosure.

**Contact:** sriharshav1@gmail.com

Please include:

1. A description of the vulnerability and its potential impact
2. Steps to reproduce
3. Any proof-of-concept code or attack scenarios you've validated
4. Your name and affiliation (if you wish to be credited)

We will acknowledge receipt within 72 hours and aim to provide an
initial assessment within 7 days.

## Disclosure Process

1. Researcher reports vulnerability privately.
2. We acknowledge receipt and begin triage.
3. We work with the researcher on a fix and coordinated disclosure
   timeline (typically 90 days, shorter for critical vulnerabilities
   with active exploitation evidence).
4. We publish a security advisory through GitHub Security Advisories
   when the fix ships.
5. We credit the researcher in the advisory (with permission).

## Scope

In scope:

- Code in this repository
- Container images published from this repository
- Documentation that affects security posture (e.g., misleading
  hardening guidance)

Out of scope:

- Vulnerabilities in third-party dependencies (please report to
  upstream; we will track CVE updates)
- Misconfigurations of operator-managed deployments
- Issues requiring physical access to the user's machine
