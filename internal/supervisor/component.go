package supervisor

// Health is the result of a component check.
type Health struct {
	OK     bool
	Detail string
}

func ok(detail string) Health  { return Health{OK: true, Detail: detail} }
func bad(detail string) Health { return Health{OK: false, Detail: detail} }

// Component is one supervised concern. Check is cheap and read-only; Repair is
// idempotent and makes the component healthy. The supervisor reconciles
// components in dependency order, repairing the first unhealthy one per tick.
type Component interface {
	Name() string
	Check() Health
	Repair() error
}
