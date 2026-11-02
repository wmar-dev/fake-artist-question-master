<!--
Sync Impact Report
Version change: template (unversioned) → 1.0.0
Modified principles: all placeholders replaced
  - [PRINCIPLE_1_NAME] → I. Well Tested
  - [PRINCIPLE_2_NAME] → II. Low Resource Usage
  - [PRINCIPLE_3_NAME] → III. Low Dependency
  - [PRINCIPLE_4_NAME] → IV. Easy to Use
Added sections: Technical Constraints, Development Workflow
Removed sections: [PRINCIPLE_5_NAME] slot (the project defines four principles)
Templates requiring updates:
  - ✅ .specify/templates/plan-template.md (Constitution Check gates)
  - ✅ .specify/templates/tasks-template.md (tests mandatory, budget verification task)
  - ✅ .claude/skills/speckit-tasks/SKILL.md (tests mandatory)
  - ✅ .specify/templates/spec-template.md (reviewed, no change needed)
  - ✅ .specify/templates/checklist-template.md (reviewed, no change needed)
  - ✅ CLAUDE.md (reviewed, no change needed)
  - ✅ .specify/templates/commands/*.md (directory does not exist; skills reviewed instead)
Follow-up TODOs: none
-->

# Fake Artist Game Constitution

## Core Principles

### I. Well Tested

- Every change to behavior MUST include automated tests in the same change.
- Game rules and state transitions MUST be covered by unit tests that run without a network,
  browser, or other external service.
- Each user story MUST have at least one automated test covering its primary acceptance
  scenario through the highest level the project can drive.
- Randomness and time MUST be injectable so that every test is deterministic.
- Every bug fix MUST include a regression test that fails without the fix.
- The full test suite MUST pass before a change is merged.

**Rationale**: A social deduction game depends on hidden information and turn order. Those
failures are hard to spot by playing and can ruin a round, so the tests have to catch them.

### II. Low Resource Usage

- Each plan MUST state numeric budgets under Constraints in Technical Context for the resources
  the feature consumes, such as memory, CPU, download size, and network traffic.
- Each budget MUST be verified by an automated test or by a measurement recorded in the change.
- Idle states MUST NOT perform periodic work, other than what a connection needs to stay alive.
- Caches, buffers, and retained game data MUST have a defined limit or expiry.

**Rationale**: The game has to run well on modest devices and cheap hosting. A stated budget
makes this a check that can fail rather than an intention.

### III. Low Dependency

- Standard library and platform features MUST be used before adding a dependency.
- Each new runtime dependency MUST be justified in the plan: what it provides, why the platform
  is not enough, and what it adds in size and transitive dependencies.
- A dependency MUST NOT be added for functionality that a small, tested module can provide.
- Dependency versions MUST be pinned, and unused dependencies MUST be removed.
- Playing the game MUST NOT require a third-party hosted service unless the plan justifies it.
- Development tools SHOULD meet the same standard.

**Rationale**: Every dependency adds upgrade work, attack surface, and resource use, and this
project is small enough that few of them pay for themselves.

### IV. Easy to Use

- Players MUST NOT need an account to play.
- Each spec MUST include a measurable success criterion for how quickly a new player can start
  or join a game.
- In every game state, the interface MUST show players whose turn it is and what they can do.
- Error messages MUST use plain language and tell the player what to do next.
- Developers MUST be able to install, run, and test the project with one documented command
  each.

**Rationale**: Friends should be able to start a round in moments, without anyone having to
explain the app first. The same applies to anyone working on the code.

## Technical Constraints

- The technology stack is chosen in the first implementation plan and MUST be the simplest one
  that satisfies these principles.
- Added complexity, such as extra projects, layers, or infrastructure, MUST be justified in the
  plan's Complexity Tracking table.

## Development Workflow

- Test tasks MUST come before the implementation tasks they cover, and the tests MUST be seen
  to fail before the implementation is written.
- Before merging, a change MUST pass the full test suite and meet the budgets in its plan.
- Each plan MUST pass the Constitution Check before research and again after design.

## Governance

This constitution supersedes other project practices. `CLAUDE.md` and the current plan provide
runtime development guidance and MUST NOT contradict it.

**Amendments**: Changes are made through `/speckit-constitution`, and MUST update the Sync Impact
Report and any affected templates in the same change.

**Versioning**: The version follows semantic versioning.

- MAJOR: A principle is removed or redefined in a backward incompatible way.
- MINOR: A principle or section is added, or guidance is materially expanded.
- PATCH: Clarifications and wording fixes with no change in meaning.

**Compliance**: Every plan and code review MUST verify compliance with this constitution. A
deviation MUST be recorded with its justification in the plan's Complexity Tracking table, and
a deviation that keeps recurring MUST be resolved by an amendment.

**Version**: 1.0.0 | **Ratified**: 2026-11-01 | **Last Amended**: 2026-11-01
