# Datamon Contributing Guide

To make the process of contributing as seamless as possible please read this
 contributing guide. Any one who follows this document can contribute to the
repo.

# Development Workflow

1. Fork [this repo](https://github.com/oneconcern/datamon/fork)
2. If an issue is not already filed for the change, define the intent of the
 work following the [Issues guide below](#Issues)
3. Start a Pull Request to bring in the work done.

# Issues

Issues are use for any new work coming in. Issues can be for

1. [New features](https://github.com/oneconcern/datamon/labels/feature-request)
2. [New uses cases within a
   feature](https://github.com/oneconcern/datamon/labels/new-use-case)
3. [Bugs](https://github.com/oneconcern/datamon/labels/bug)

Each issue should articulate for other contributors and maintainers.

1. Why is it an issue?
2. What is the planned change for it?
3. What is the scope and definition of success?
4. What is not part of the needed resolution of the issue?

The motivation here is to clearly articulate the need, the form and details of
an issue before a Pull Request is posted that addresses the issue. This is to insure
that the other reviewers can appreciate the need for the Pull Request and do
justice to the code review.

Large feature requests issues can spawn other focused issues that
 complete one end to end functionality.

# Milestones and tracking

The master is always shippable with a clear set of functionality that works.
Milestones are analogous to sprints and a release will be cut after every
milestone. Milestones are planned on a 3 week cadence.
Large releases that are a culmination of a few milestones are tracked as
[projects](https://github.com/oneconcern/datamon/projects).

## Planning

In the weeks leading up to a Milestone, the issues planned for it need to be
reviewed to make sure that they comply with the requirements listed above. Only
issues that meet the requirement will be worked on and reviewed to close a
milestone.

Maintainers need to meet through the week to insure that the backlog of issues
for upcoming sprints has been fleshed out to be actionable.

# WIP: Dealing with incomplete work

It is encouraged to use Pull Requests when incomplete work needs to be reviewed. Adding [WIP](https://github.com/apps/wip) to the title of a commit or to 
the title of the Pull Request, the Pull Request can be reviewed but will be blocked from being merged in.

# Git

This repo requires work to be signed off with a verifiable GPG signature that the work is compliant with [Digital Certificate of Origin](https://developercertificate.org).

## Basic setup

A valid and appropriate email address is needed.

```
git config --global user.useConfigOnly true
git config --global --unset-all user.email
```

## Authenticity of commits, GPG

This repo uses [PGP
keys](https://blog.github.com/2016-04-05-gpg-signature-verification/). Please
setup PGP before pushing changes.
One option to simplify the PGP setup is to use a service such as
[KeyBase](https://github.com/pstadler/keybase-gpg-github)

## DCO

This repo mandates [DCO](https://github.com/apps/dco). Developers submitting the change have to comply with the DCO text and sign off commits.

# Testing

## Unit Tests
Sufficient unit tests need to be written to cover edge conditions.
## End to End Tests
Every commit goes through the entire set of end to end tests before it makes it
way to the target branch.
## Circle CI
[TODO: Circle CI Integration](https://github.com/oneconcern/datamon/issues/7)
