// Package awsdoctor wraps the aws-doctor CLI tool as a subprocess,
// parsing its JSON output into typed structs. aws-doctor is CLI-only
// (not importable as a library), so we shell out via os/exec.
package awsdoctor

// RunOpts configures an aws-doctor invocation.
type RunOpts struct {
	Profile string
	Region  string
}

// WasteReport is the parsed JSON output of `aws-doctor --waste --output json`.
// Field names match the real aws-doctor output schema.
type WasteReport struct {
	AccountID           string             `json:"account_id"`
	GeneratedAt         string             `json:"generated_at"`
	HasWaste            bool               `json:"has_waste"`
	UnusedElasticIPs    []ElasticIP        `json:"unused_elastic_ips"`
	UnusedEBSVolumes    []EBSVolume        `json:"unused_ebs_volumes"`
	StoppedVolumes      []EBSVolume        `json:"stopped_instance_volumes"`
	StoppedInstances    []StoppedInstance  `json:"stopped_instances"`
	ReservedInstances   []ReservedInstance `json:"reserved_instances"`
	UnusedLoadBalancers []LoadBalancer     `json:"unused_load_balancers"`
	UnusedAMIs          []AMI              `json:"unused_amis"`
	OrphanedSnapshots   []Snapshot         `json:"orphaned_snapshots"`
	StaleSnapshots      []Snapshot         `json:"stale_snapshots"`
	UnusedKeyPairs      []KeyPair          `json:"unused_key_pairs"`
}

// ElasticIP is an unused Elastic IP finding.
type ElasticIP struct {
	PublicIP     string `json:"public_ip"`
	AllocationID string `json:"allocation_id"`
}

// EBSVolume is an unattached or stopped-instance EBS volume.
type EBSVolume struct {
	VolumeID string `json:"volume_id"`
	SizeGiB  int32  `json:"size_gib"`
	Status   string `json:"status"`
}

// StoppedInstance is a long-stopped EC2 instance.
type StoppedInstance struct {
	InstanceID string `json:"instance_id"`
	StoppedAt  string `json:"stopped_at,omitempty"`
	DaysAgo    int    `json:"days_ago,omitempty"`
}

// ReservedInstance is an expiring or unused RI.
type ReservedInstance struct {
	ReservedInstanceID string `json:"reserved_instance_id"`
	InstanceType       string `json:"instance_type"`
	ExpirationDate     string `json:"expiration_date"`
	DaysUntilExpiry    int    `json:"days_until_expiry"`
	State              string `json:"state"`
	Status             string `json:"status"`
}

// LoadBalancer is an unused ELB/ALB/NLB.
type LoadBalancer struct {
	Name string `json:"name"`
	ARN  string `json:"arn"`
	Type string `json:"type"`
}

// AMI is an unused AMI with potential savings.
type AMI struct {
	ImageID            string  `json:"image_id"`
	Name               string  `json:"name"`
	Description        string  `json:"description,omitempty"`
	CreationDate       string  `json:"creation_date"`
	DaysSinceCreate    int     `json:"days_since_create"`
	IsPublic           bool    `json:"is_public"`
	SnapshotSizeGB     int64   `json:"snapshot_size_gb"`
	MaxPotentialSaving float64 `json:"max_potential_saving_monthly"`
}

// Snapshot is an orphaned or stale EBS snapshot.
type Snapshot struct {
	SnapshotID          string  `json:"snapshot_id"`
	VolumeID            string  `json:"volume_id,omitempty"`
	VolumeExists        bool    `json:"volume_exists"`
	UsedByAMI           bool    `json:"used_by_ami"`
	SizeGB              int32   `json:"size_gb"`
	StartTime           string  `json:"start_time"`
	DaysSinceCreate     int     `json:"days_since_create"`
	Category            string  `json:"category"`
	Reason              string  `json:"reason"`
	MaxPotentialSavings float64 `json:"max_potential_savings"`
}

// KeyPair is an unused SSH key pair.
type KeyPair struct {
	KeyName         string `json:"key_name"`
	KeyPairID       string `json:"key_pair_id"`
	CreationDate    string `json:"creation_date"`
	DaysSinceCreate int    `json:"days_since_create"`
}

// TrendReport is the parsed JSON output of `aws-doctor --trend --output json`.
type TrendReport struct {
	AccountID   string      `json:"account_id"`
	GeneratedAt string      `json:"generated_at"`
	Months      []MonthCost `json:"months"`
}

// MonthCost is one month's total cost in a trend report.
type MonthCost struct {
	Start string  `json:"start"`
	End   string  `json:"end"`
	Total float64 `json:"total"`
	Unit  string  `json:"unit"`
}
