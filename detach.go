package main

// Windows process creation flags, spelled out here rather than imported so the
// order below can be tested on any machine.
const (
	createBreakawayFromJob = 0x01000000
	detachedProcess        = 0x00000008
	createNewProcessGroup  = 0x00000200
)

/* How to start the helper that replaces the package, in order of preference.
 *
 * Replacing a package with ForceTargetApplicationShutdown terminates the whole
 * app container, and a process this app started is inside that container. The
 * helper would be killed part way through its own work, leaving the app shut
 * down and nothing started in its place. Breaking out of the job is what keeps
 * it alive long enough to finish.
 *
 * A job that forbids breakaway refuses the flag outright, so the second entry
 * is the same thing without it: worse odds, but the alternative is not starting
 * the helper at all. */
func detachFlagOrder() []uint32 {
	return []uint32{
		createBreakawayFromJob | detachedProcess | createNewProcessGroup,
		detachedProcess | createNewProcessGroup,
	}
}
