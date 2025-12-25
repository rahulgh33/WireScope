package plugin
package plugin

// Plugin system for custom metric collectors and notification channels








































}	return c, ok	c, ok := r.channels[name]func (r *Registry) GetChannel(name string) (NotificationChannel, bool) {}	return c, ok	c, ok := r.collectors[name]func (r *Registry) GetCollector(name string) (MetricCollector, bool) {}	r.channels[c.Name()] = cfunc (r *Registry) RegisterChannel(c NotificationChannel) {}	r.collectors[c.Name()] = cfunc (r *Registry) RegisterCollector(c MetricCollector) {}	}		channels:   make(map[string]NotificationChannel),		collectors: make(map[string]MetricCollector),	return &Registry{func NewRegistry() *Registry {}	channels   map[string]NotificationChannel	collectors map[string]MetricCollectortype Registry struct {}	Send(message string) error	Name() stringtype NotificationChannel interface {}	Collect() (map[string]float64, error)	Name() stringtype MetricCollector interface {