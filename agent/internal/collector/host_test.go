package collector

import "testing"

func TestParseMeminfo(t *testing.T) {
	content := []byte("MemTotal:       16384256 kB\nMemAvailable:    8192128 kB\n")
	totalKB, availableKB := parseMeminfo(content)

	if totalKB != 16384256 {
		t.Fatalf("expected total mem 16384256, got %f", totalKB)
	}
	if availableKB != 8192128 {
		t.Fatalf("expected available mem 8192128, got %f", availableKB)
	}
}

func TestParseOSRelease(t *testing.T) {
	content := []byte("NAME=\"Ubuntu\"\nVERSION=\"24.04.2 LTS (Noble Numbat)\"\nPRETTY_NAME=\"Ubuntu 24.04.2 LTS\"\n")
	value := parseOSRelease(content, "PRETTY_NAME")

	if value != "Ubuntu 24.04.2 LTS" {
		t.Fatalf("expected PRETTY_NAME to be parsed, got %q", value)
	}
}

func TestParseCPUInfo(t *testing.T) {
	content := []byte(`
processor   : 0
physical id : 0
core id     : 0
model name  : AMD EPYC Processor

processor   : 1
physical id : 0
core id     : 1
model name  : AMD EPYC Processor

processor   : 2
physical id : 0
core id     : 0
model name  : AMD EPYC Processor

processor   : 3
physical id : 0
core id     : 1
model name  : AMD EPYC Processor
`)

	info := parseCPUInfo(content)
	if info.model != "AMD EPYC Processor" {
		t.Fatalf("expected cpu model to be parsed, got %q", info.model)
	}
	if info.threads != 4 {
		t.Fatalf("expected 4 cpu threads, got %d", info.threads)
	}
	if info.cores != 2 {
		t.Fatalf("expected 2 cpu cores, got %d", info.cores)
	}
}

func TestCalculateCPUUsage(t *testing.T) {
	usage := calculateCPUUsage(
		cpuCounters{idle: 100, total: 200},
		cpuCounters{idle: 160, total: 320},
	)

	if usage < 49.9 || usage > 50.1 {
		t.Fatalf("expected cpu usage close to 50%%, got %f", usage)
	}
}

func TestParseNetworkAndDiskCounters(t *testing.T) {
	network := parseNetworkCounters([]byte(`
Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 100 0 0 0 0 0 0 0 200 0 0 0 0 0 0 0
  eth0: 1024 0 0 0 0 0 0 0 2048 0 0 0 0 0 0 0
`))
	if network.rxBytes != 1024 || network.txBytes != 2048 {
		t.Fatalf("expected network counters to ignore loopback, got rx=%d tx=%d", network.rxBytes, network.txBytes)
	}

	disk := parseDiskCounters([]byte(`
   8       0 sda 100 0 200 0 100 0 400 0 0 0 0 0 0 0 0 0 0
   8       1 sda1 50 0 100 0 50 0 200 0 0 0 0 0 0 0 0 0 0
 259       0 nvme0n1 10 0 50 0 10 0 70 0 0 0 0 0 0 0 0 0 0
`), map[string]struct{}{
		"sda":     {},
		"nvme0n1": {},
	})

	expectedRead := uint64(250 * diskSectorSizeBytes)
	expectedWrite := uint64(470 * diskSectorSizeBytes)
	if disk.readBytes != expectedRead || disk.writeBytes != expectedWrite {
		t.Fatalf("expected whole-disk counters only, got read=%d write=%d", disk.readBytes, disk.writeBytes)
	}
}

func TestHostPathUsesMountedHostRootWhenConfigured(t *testing.T) {
	if got := hostPath("", "/proc/stat"); got != "/proc/stat" {
		t.Fatalf("expected default host path to stay unchanged, got %q", got)
	}
	if got := hostPath("/hostfs", "/proc/stat"); got != "/hostfs/proc/stat" {
		t.Fatalf("expected mounted host path, got %q", got)
	}
	if got := hostPath("/hostfs", "/"); got != "/hostfs" {
		t.Fatalf("expected host root path, got %q", got)
	}
}
