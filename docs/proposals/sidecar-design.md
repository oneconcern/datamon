# Dataon sidecar

## Needed longer term corrective actionss

* [ ] Make this a more compact binary, with 12 factor-app parameterization, which is difficult to achieve in shell and easy in go
* [ ] Sidecar_param binary is essentially overlapping with sp13/viper: a golang-based sidecar wouldn't need that, just viper
* [ ] Adapt wrapper logic to support many sidecars, including a mix of fuse & postgres ones
* [ ] Adapt wrapper logic to support other workflows, such as "only read, don't update"
* [ ] Take actual provisions to handle postgres version migrations
* [ ] Handle errors gracefully & allow for out of band signaling: the "application wrapper" should stop when sidecars fail

## Alternative design proposal

### Primer

This is a proposal for a drop-in replacement of the current sidecars.

### Principles

1. A sidecar is a ready-to-use container image. The smaller the better.
2. It should be simple to configure and run.
3. It should properly handle errors and interrupts the global worklow upon error
4. We should be able to test it unitarily, without the full fledged k8 demo apparatus
5. Users should be able to opt in for fuse, or populate a volume with download
6. We application wrapper/sidecar pair should be decoupled to adapt to new data workflows, possibly on different pods

### Proposal
1. Ship sidecar functionality with datamon binary (e.g. `datamon sidecar ...`). The functionality is implemented in golang.
   Sidecar is a statically linked binary, but we still need a more complete distribution around (for fuse, for postgres).
   Experiment with alpine to figure out a minimal image which is quickly downloaded.
2. Sidecar supports parameterization from: config (e.g. configmap), env and flags (basically, use viper). Sensible defaults are set.
   All datamon flags should be similarly configurable in a sensible way (e.g. to configure cache, metrics, etc).
3. 

### Detailed design

#### Consequences
* Package `pkg/sidecar/param` is reworked: we only keep data structures and drop the unmarshalling logic
* `hack/fuse-demo` now contains only demo scripts, and no actually released code
* Postgres commands are packaged via 
