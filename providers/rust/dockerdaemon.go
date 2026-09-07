package rust

import "github.com/pablogore/shipwright/providers/rust/daggerkit"

// dockerDindImage is the official Docker-in-Docker image used to give a
// RustCommand/RustIntegrationTester container access to a real, privileged
// Docker daemon.
//
// This exists in place of the more obvious approach — mounting the host's
// own Docker socket into the container (Docker-outside-of-Docker) — which
// was this package's original mechanism and does not work through Dagger.
// Dagger always executes a container through its own nested BuildKit
// runtime, so a container that mounts the host daemon's socket can issue
// Docker API calls (e.g. "start a Postgres container") but has no network
// route to that daemon's own bridge network. testcontainers-rs resolves a
// sibling container's published ports by looking up the daemon's bridge
// network gateway address (testcontainers' client.rs::docker_hostname()),
// which is unreachable from inside a Dagger-managed container — confirmed
// against a real ego-rs run-suite invocation, which got as far as asking
// the daemon to start Postgres and then timed out trying to connect to it.
// This isn't specific to any one host's Docker setup (verified against
// colima locally): it follows from Dagger's own container sandboxing model,
// which deliberately exposes no host-network passthrough.
//
// The fix is the standard "docker:dind" CI sidecar pattern instead: run a
// real, privileged dockerd as a Dagger Service and bind it into the caller's
// container under a network alias — a routable connection Dagger itself
// establishes between the two containers it orchestrates, unlike a socket
// mount. Pointing DOCKER_HOST at that alias over TCP works because
// testcontainers-rs's docker_hostname() takes a different branch for a TCP
// DOCKER_HOST: it returns the URL's own host component (the alias) directly,
// with no bridge-gateway lookup at all, so the alias resolves over Dagger's
// service-binding network exactly like any other container-to-container
// dependency in a Dagger pipeline.
const dockerDindImage = "docker:27-dind"

// dockerServiceAlias is the network alias the DinD daemon is bound under,
// and dockerHostEnv is DOCKER_HOST's value pointing at it — the client
// library used by both providers, testcontainers-rs, reads this exact
// environment variable.
const (
	dockerServiceAlias = "docker"
	dockerHostEnv      = "tcp://" + dockerServiceAlias + ":2375"
)

// dockerdCommand is dockerd's own startup command, passed as AsService's
// Args rather than a prior WithExec: WithExec's semantics are "run this to
// completion and snapshot the result," which never happens for a daemon
// that runs forever — chaining WithExec().AsService() leaves the container
// permanently pending, and the AsService call itself never resolves.
// AsService's own Args/InsecureRootCapabilities options run the command as
// the service's live process instead, exactly like the base dind image's
// own default entrypoint would. Confirmed against a real, isolated dagger
// probe: WithExec().AsService() hung indefinitely (30m timeout) even though
// dockerd itself was up and healthy within 2 seconds; AsService(Args:...)
// resolved in ~2s end to end.
var dockerdCommand = []string{"dockerd", "--host=tcp://0.0.0.0:2375", "--tls=false"}

// withDockerDaemon attaches a privileged Docker-in-Docker service to
// container under dockerServiceAlias and points DOCKER_HOST at it, giving
// the container a real, network-reachable Docker daemon.
func withDockerDaemon(client daggerkit.DaggerClient, container daggerkit.DaggerContainer) daggerkit.DaggerContainer {
	dind := client.Container().
		From(dockerDindImage).
		WithExposedPort(2375).
		AsService(dockerdCommand)

	return container.
		WithServiceBinding(dockerServiceAlias, dind).
		WithEnvVariable("DOCKER_HOST", dockerHostEnv)
}
