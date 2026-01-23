# Terraform Provider TrueNAS

This repository contains a Terraform provider for TrueNAS SCALE and Community editions. It uses the `midclt` command to communicate with the TrueNAS API.

## Development workflow

1. Development tasks are conducted using `mise`. Run `mise tasks` to see what tasks are available.

### Design and implementation plans

When asked to write an implementation plan, the context should include the current code coverage from `mise run coverage`. In the verification tasks, verify that the code coverage has either improved or maintained with the baseline. 

## Ethos

- Always write idiomatic terraform.
- Strive for 100% code coverage where possible.
