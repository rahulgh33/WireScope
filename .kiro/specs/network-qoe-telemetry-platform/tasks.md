# Implementation Plan

## Overview

This implementation plan converts the Network QoE Telemetry Platform design into a series of incremental coding tasks organized into three milestones. Each milestone delivers a working system with increasing sophistication, allowing for early demos and iterative development.

## Milestone Structure

**Milestone A: End-to-End Data Path** - Core functionality for "it works" demo
**Milestone B: Correctness & Reliability** - Production-grade reliability patterns  
**Milestone C: Observability & Operations** - Full production observability stack

## Task List

### Milestone A: End-to-End Data Path

- [x] 1. Set up project structure and development environment
  - Create Go module structure with cmd/, internal/, pkg/, and config/ directories
  - Set up Docker Compose with PostgreSQL, NATS JetStream, Prometheus, Grafana, Jaeger, and OpenTelemetry Collector
  - Create Makefile with standard targets (up, down, migrate, build, test)
  - Initialize database migration system
  - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 9.1_

- [x] 2. Implement database schema and migrations
  - Create initial migration for events_seen deduplication table with UUID primary key
  - Create migration for agg_1m aggregates table with composite primary key and quality counters
  - Add database indexes for query optimization (client_id, target, window_start_ts)
  - Implement database connection utilities with connection pooling
  - _Requirements: 11.1, 11.3, 11.4_

- [ ] 3. Create core data models and interfaces
  - Define TelemetryEvent struct with schema versioning and recv_ts_ms field
  - Implement NetworkContext and TimingMeasurements structs
  - Create WindowedAggregate struct with quality counters and error stage counts
  - Define EventProcessor interface for message queue operations
  - Implement validation functions for data integrity
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_

- [ ] 3.1 Write property test for event structure validation
  - **Property 2: Event structure validity**
  - **Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5**

- [ ] 4. Implement NATS JetStream integration
  - Set up NATS JetStream client with stream and consumer configuration
  - Create telemetry-events stream with file storage and retention policies
  - Create telemetry-events-dlq stream for poison message handling
  - Implement EventProcessor with publish, consume, and acknowledgment methods
  - _Requirements: 3.1, 8.4_

- [ ] 5. Create test target server for deterministic testing
  - Build HTTP server with /health endpoint (fast TTFB)
  - Add /slow?ms=N endpoint with controllable delay
  - Implement /fixed/1mb.bin endpoint for throughput testing
  - Add optional TLS endpoint for secure connection testing
  - Configure as Docker service in compose stack
  - _Requirements: Testing infrastructure_

- [ ] 6. Build probe agent CLI (minimal)
  - Implement network measurement functions for DNS, TCP, TLS, HTTP timings
  - Create throughput measurement with 1MB downloads, Cache-Control headers, and fresh connections
  - Add stable client_id generation and local storage
  - Create configuration management for targets, intervals, and API credentials
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

- [x] 7. Create ingest API service (minimal)
  - Implement HTTP server with basic authentication middleware using API tokens
  - Add request validation with schema version checking and forward compatibility
  - Implement event publishing to NATS JetStream with error handling
  - Add recv_ts_ms timestamp injection for clock skew debugging
  - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

- [x] 8. Implement aggregator consumer with deduplication
  - Create NATS consumer with durable subscription and explicit acknowledgment
  - Implement exactly-once aggregate effects using events_seen deduplication table
  - Build transactional pattern: INSERT INTO events_seen ON CONFLICT DO NOTHING
  - Add window assignment logic using FLOOR(ts_ms / 60000) * 60000
  - Store raw samples in memory per window per (client_id, target)
  - _Requirements: 3.2, 3.3, 3.5, 4.1_

- [x] 9. Build percentile calculation and aggregate persistence
  - Implement exact P50/P95 calculation from sorted arrays (up to 10,000 samples)
  - Add uniform downsampling for windows exceeding sample limits
  - Implement quality counters: count_total, count_success, count_error, per-stage error counts
  - Build aggregate upsert logic with conflict resolution
  - _Requirements: 4.2, 4.4, 4.5_

- [x] 10. Milestone A Demo - End-to-end data flow
  - Verify probe → ingest → JetStream → aggregator → PostgreSQL pipeline
  - Test basic percentile calculations and quality counters
  - Demonstrate window-based aggregation with sample data

### Milestone B: Correctness & Reliability

- [x] 11. Add late event handling and DLQ republish logic
  - Implement late event handling with 2-minute tolerance based on processing time
  - Add DLQ republish logic on final delivery attempt failure
  - Create poison message handling with max delivery checks
  - _Requirements: 3.4, 8.4_

- [x] 12. Implement backpressure mechanisms
  - Add bounded local queue in probe with exponential backoff
  - Implement rate limiting per client_id in ingest API (simple in-memory token bucket)
  - Create in-flight message limits in aggregator with proper acknowledgment
  - _Requirements: 8.2, 8.3, 8.4_

- [x] 13. Property testing hardening pass
  - Write integration property test for exactly-once aggregate effects via dedup
  - Create property test for transactional consistency (aggregator only ACKs after DB commit)
  - Add property test for late event handling with explicit time reference
  - Implement property test for window assignment accuracy
  - Test percentile calculation correctness (scoped to ≤10k samples for exactness)
  - _Properties: 3, 4, 5, 6, 7_

- [ ] 14. Add basic observability (core 6 metrics)
  - Implement ingest request rate and error rate metrics
  - Add queue lag monitoring
  - Create processing delay histogram (end-to-end latency)
  - Add events_processed_total split by success/duplicate/error
  - Implement dedup_rate metric
  - Add dlq_messages_total counter
  - _Requirements: 6.1, 6.2, 6.3_

- [ ] 15. Milestone B Demo - Reliability validation
  - Test duplicate event handling with unchanged aggregates
  - Demonstrate aggregator restart with safe continuation
  - Show backpressure activation and recovery
  - Validate DLQ routing for poison messages

### Milestone C: Observability & Operations

- [ ] 16. Implement diagnosis engine with explicit thresholds
  - Create baseline calculation using simple moving average over last 10 windows
  - Implement DNS-bound diagnosis: DNS p95 ≥60% of total latency p95 and exceeds baseline by ≥50%
  - Add Handshake-bound diagnosis: TCP/TLS p95 exceeds baseline by 2σ or 100%
  - Create Server-bound diagnosis: TTFB p95 exceeds baseline by 2σ while connections normal
  - Implement Throughput-bound diagnosis: throughput p50 drops ≥30% below baseline
  - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

- [ ]* 16.1 Write property tests for diagnosis thresholds
  - **Properties 9-13: Diagnosis accuracy with explicit thresholds**
  - **Validates: Requirements 5.1, 5.2, 5.3, 5.4, 5.5**

- [ ] 17. Add comprehensive observability
  - Expand Prometheus metrics with cardinality management (hash client_id, target)
  - Create detailed ingest API metrics: authentication failures, rate limiting
  - Add comprehensive queue metrics: consumer lag, ack pending, processing duration
  - Implement aggregator metrics: window updates, percentile computation time
  - Add database metrics: connections, query duration, transaction time, table sizes
  - _Requirements: 6.1, 6.2, 6.3_

- [ ] 18. Implement OpenTelemetry distributed tracing
  - Set up OpenTelemetry SDK with Jaeger exporter configuration
  - Add tracing spans for ingest API requests with context propagation
  - Implement aggregator operation spans with database transaction tracing
  - Create probe measurement spans with network operation details
  - Add span attributes for debugging: client_id, target, window_start_ts, error_stage
  - _Requirements: 6.4, 6.5_

- [ ] 19. Create Grafana dashboards and alerting
  - Build starter Grafana dashboard JSON with network performance visualizations
  - Create dashboard panels for percentile trends, error rates, and diagnosis labels
  - Add alerting rules for SLO violations (99% of events processed within 10 seconds)
  - Implement threshold-based alerts for diagnosis patterns and system health
  - _Requirements: 9.2, 9.5, 11.5_

- [ ] 20. Implement failure-mode testing scripts
  - Create integration test for duplicate event publishing with unchanged aggregates
  - Build aggregator restart test with safe continuation and no data loss
  - Implement burst traffic test with lag monitoring and recovery validation
  - Add poison message test with DLQ routing and manual replay
  - Create database failure simulation with transaction retry behavior
  - _Requirements: 8.1, 8.5_

- [ ] 21. Add database maintenance and retention
  - Implement daily cleanup job for events_seen table (delete >7 days)
  - Create weekly partition cleanup for agg_1m table (remove >90 days)
  - Add automated partition creation for upcoming months
  - Implement database health checks and connection monitoring
  - _Requirements: 11.1_

- [ ] 22. Final integration and documentation
  - Create comprehensive README.md with architecture diagram and setup instructions
  - Write step-by-step demo instructions with sample probe configurations
  - Document API endpoints with OpenAPI specification
  - Create troubleshooting guide for common operational issues
  - Add performance tuning recommendations for production deployment
  - _Requirements: 9.2, 9.3, 9.4_

- [ ] 23. Final Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

### Milestone D: AI-Powered Analytics (Future Enhancement)

- [ ] 24. Design AI agent architecture for network telemetry analysis
  - Define agent capabilities: pattern detection, anomaly identification, root cause analysis
  - Design conversational interface for querying network data
  - Plan integration with existing PostgreSQL aggregates and time-series data
  - Document use cases: trend analysis, performance degradation detection, capacity planning
  - _Requirements: Future feature - AI-powered insights_

- [ ] 25. Implement AI agent data access layer
  - Create specialized SQL queries for common analysis patterns
  - Build time-series data aggregation for trend analysis
  - Implement multi-dimensional filtering (client, target, time range, error types)
  - Add data summarization APIs for LLM consumption
  - Create context-aware query optimization for large datasets
  - _Requirements: Future feature - Data layer for AI agent_

- [ ] 26. Build AI agent core functionality
  - Implement natural language query understanding
  - Create response generation for network insights
  - Add pattern recognition for common network issues
  - Implement anomaly detection across time windows
  - Build comparative analysis (baseline vs current performance)
  - Add visualization recommendation engine
  - _Requirements: Future feature - AI agent intelligence_

- [ ] 27. Create AI agent interface and integration
  - Build conversational API endpoint for agent queries
  - Implement chat history and context management
  - Add authentication and user session handling
  - Create example prompts and guided workflows
  - Integrate with existing Grafana dashboards
  - Build CLI tool for interactive agent queries
  - _Requirements: Future feature - AI agent interface_

- [ ] 28. Add advanced AI agent features
  - Implement predictive analytics for capacity planning
  - Add automated report generation for network health
  - Create proactive alerting based on learned patterns
  - Implement comparative analysis across client segments
  - Add correlation detection between network events
  - Build recommendation engine for optimization opportunities
  - _Requirements: Future feature - Advanced AI capabilities_

- [ ] 29. Milestone D Demo - AI-Powered Network Analysis
  - Demonstrate natural language queries: "Which clients had the worst performance today?"
  - Show trend analysis: "Is DNS performance degrading over the past week?"
  - Demonstrate anomaly detection: "Identify unusual latency patterns"
  - Show root cause analysis: "Why is throughput low for client X?"
  - Present predictive insights: "Forecast next week's capacity needs"
  - Validate comparative analysis: "Compare weekend vs weekday performance"

### Milestone E: User Interface & Visualization

- [ ] 30. Design UI architecture and technology stack
  - Select frontend framework (React, Vue, or Svelte)
  - Design REST API endpoints for UI data access
  - Plan responsive layout with mobile support
  - Define user authentication and authorization strategy
  - Create wireframes for key screens: dashboard, clients, targets, diagnostics
  - _Requirements: UI foundation_

- [ ] 31. Implement backend API for UI
  - Create REST API endpoints for aggregate queries (time-series data)
  - Add client management APIs: list, filter, search clients
  - Implement target management APIs: list targets, view target metrics
  - Build diagnostic history API with filtering and pagination
  - Add real-time metrics API with WebSocket support
  - Implement authentication endpoints (login, logout, session management)
  - _Requirements: API layer for UI_

- [ ] 32. Build main dashboard view
  - Create overview dashboard with key metrics summary
  - Implement time-series charts for P50/P95 latency trends
  - Add error rate visualization with breakdown by stage
  - Build throughput trends with client/target filtering
  - Create active clients and targets summary cards
  - Add recent diagnostics feed with severity indicators
  - Implement time range selector (1h, 6h, 24h, 7d, 30d, custom)
  - _Requirements: Dashboard visualization_

- [ ] 33. Create client and target management views
  - Build clients list with search, sort, and filter capabilities
  - Implement client detail view with performance history
  - Create target list with health status indicators
  - Add target detail view with per-client performance breakdown
  - Implement comparison view for multiple clients or targets
  - Add export functionality for reports (CSV, JSON)
  - _Requirements: Client/target management UI_

- [ ] 34. Implement diagnostics and analysis views
  - Create diagnostic dashboard with problem categorization
  - Build detailed diagnostic view with root cause analysis
  - Implement baseline comparison visualization
  - Add alert configuration interface for thresholds
  - Create historical diagnostics timeline view
  - Implement diagnostic filtering by client, target, and type
  - _Requirements: Diagnostics UI_

- [ ] 35. Add real-time monitoring and live updates
  - Implement WebSocket connection for live metric updates
  - Create live dashboard with auto-refreshing charts
  - Add real-time event stream viewer
  - Build live probe status indicators
  - Implement notification system for critical alerts
  - Add connection status indicator with auto-reconnect
  - _Requirements: Real-time features_

- [ ] 36. Build configuration and admin views
  - Create probe configuration interface
  - Add ingest API token management
  - Implement user management and permissions
  - Build system settings view (retention, thresholds)
  - Create database maintenance controls
  - Add system health monitoring dashboard
  - _Requirements: Admin interface_

- [ ] 37. Implement UI testing and polish
  - Add unit tests for UI components
  - Create integration tests for API interactions
  - Implement responsive design testing
  - Add accessibility features (WCAG compliance)
  - Create loading states and error handling
  - Add tooltips and help documentation
  - Implement dark mode theme
  - _Requirements: UI quality and testing_

- [ ] 38. Milestone E Demo - Complete UI Experience
  - Demonstrate dashboard with live telemetry data
  - Show client performance analysis workflow
  - Walk through diagnostic investigation process
  - Demonstrate real-time monitoring capabilities
  - Show configuration and admin features
  - Validate responsive design on different devices

### Milestone F: Production Readiness & Customization

- [ ] 39. Add licensing and open source preparation
  - Add LICENSE file (MIT or Apache 2.0)
  - Create CONTRIBUTING.md with contribution guidelines
  - Add CODE_OF_CONDUCT.md for community standards
  - Update README with proper attribution and credits
  - Add SECURITY.md for vulnerability reporting
  - _Requirements: Open source compliance_

- [ ] 40. Implement configuration management system
  - Replace CLI flags with YAML/JSON config files
  - Add support for environment variables (.env files)
  - Implement configuration validation and defaults
  - Create config examples for different deployment scenarios
  - Add hot-reload capability for configuration changes
  - Document all configuration options
  - _Requirements: Deployment flexibility_

- [ ] 41. Build multi-tenancy and authentication
  - Add organization/account separation layer
  - Implement OAuth2/JWT authentication
  - Create role-based access control (RBAC)
  - Add API key management per organization
  - Implement tenant isolation in database
  - Create tenant-specific quotas and limits
  - _Requirements: Enterprise multi-tenancy_

- [ ] 42. Add deployment automation and orchestration
  - Create Helm charts for Kubernetes deployment
  - Build Docker Compose production configuration
  - Add Terraform modules for cloud infrastructure
  - Implement health checks and readiness probes
  - Create deployment scripts for common platforms (AWS, GCP, Azure)
  - Add migration and upgrade guides
  - _Requirements: Production deployment_

- [ ] 43. Implement plugin and extension system
  - Design plugin architecture with defined interfaces
  - Create custom aggregation function plugins
  - Add custom metric collectors
  - Build notification channel plugins (Slack, PagerDuty, email)
  - Implement custom storage backend adapters
  - Create plugin marketplace documentation
  - _Requirements: Extensibility_

- [ ] 44. Add alternative storage backends
  - Implement TimescaleDB adapter for time-series optimization
  - Add ClickHouse adapter for analytics workloads
  - Create S3/Object storage adapter for archival
  - Build Prometheus remote write integration
  - Add data export utilities (CSV, Parquet, JSON)
  - Document storage backend selection guide
  - _Requirements: Storage flexibility_

- [ ] 45. Build enterprise integrations
  - Add Prometheus metrics exporter
  - Create Grafana plugin/datasource
  - Implement OpenTelemetry integration
  - Add webhook notification system
  - Create REST API for external integrations
  - Build client SDKs (Python, JavaScript, Go)
  - _Requirements: Enterprise ecosystem_

- [ ] 46. Add advanced deployment features
  - Implement horizontal scaling for aggregators
  - Add load balancing for ingest API
  - Create distributed caching layer
  - Implement backup and disaster recovery
  - Add cross-region replication
  - Create performance tuning guides
  - _Requirements: Scalability_

- [ ] 47. Create comprehensive documentation
  - Write architecture documentation with diagrams
  - Create API reference documentation
  - Build deployment guide for various platforms
  - Write customization and extension guides
  - Create troubleshooting and FAQ sections
  - Add video tutorials and demos
  - Build example use cases and templates
  - _Requirements: User enablement_

- [ ] 48. Milestone F Demo - Production Deployment
  - Demonstrate multi-tenant deployment
  - Show Kubernetes deployment with Helm
  - Validate plugin system with custom extensions
  - Demonstrate cross-platform compatibility
  - Show backup and recovery procedures
  - Present documentation and onboarding experience