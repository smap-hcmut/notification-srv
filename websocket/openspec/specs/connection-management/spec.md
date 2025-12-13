# Connection Management and Authorization

## ADDED Requirements

### Requirement: Topic-Based Connection Creation MUST Be Supported
WebSocket connections MUST be created with topic subscription filters based on query parameters.

#### Scenario: Project Connection Creation
**Given** an authenticated user makes a WebSocket connection request to `ws://localhost:8081/ws?projectId=proj_123`  
**When** the connection is being established  
**Then** a Connection object must be created with projectID set to "proj_123"  
**And** jobID must remain empty  
**And** the connection must be registered in the Hub with topic filters

#### Scenario: Job Connection Creation
**Given** an authenticated user makes a WebSocket connection request to `ws://localhost:8081/ws?jobId=job_789`  
**When** the connection is being established  
**Then** a Connection object must be created with jobID set to "job_789"  
**And** projectID must remain empty  
**And** the connection must be registered in the Hub with topic filters

#### Scenario: General Connection Creation
**Given** an authenticated user makes a WebSocket connection request to `ws://localhost:8081/ws` without topic parameters  
**When** the connection is being established  
**Then** a Connection object must be created with both projectID and jobID empty  
**And** the connection must receive all messages for that user (backward compatibility)

#### Scenario: Multiple Topic Parameters Handling
**Given** a connection request with both `projectId=proj_123` and `jobId=job_789`  
**When** processing the connection parameters  
**Then** it must reject the connection with HTTP 400 status  
**And** must return error message "cannot subscribe to multiple topic types simultaneously"

### Requirement: Authorization Validation for Topic Access SHALL Be Enforced
All topic-specific connections MUST validate user access permissions before establishment.

#### Scenario: Valid Project Access Authorization
**Given** user "user_456" requests connection to `projectId=proj_123`  
**And** the user has valid access to project "proj_123"  
**When** validating authorization  
**Then** the connection must be allowed to proceed  
**And** must be established with project topic filter

#### Scenario: Invalid Project Access Authorization
**Given** user "user_456" requests connection to `projectId=proj_999`  
**And** the user does not have access to project "proj_999"  
**When** validating authorization  
**Then** the connection must be rejected with HTTP 403 status  
**And** must return error message "unauthorized access to project: proj_999"  
**And** must log the authorization failure attempt

#### Scenario: Valid Job Access Authorization  
**Given** user "user_456" requests connection to `jobId=job_789`  
**And** the user has valid access to job "job_789"  
**When** validating authorization  
**Then** the connection must be allowed to proceed  
**And** must be established with job topic filter

#### Scenario: Invalid Job Access Authorization
**Given** user "user_456" requests connection to `jobId=job_999`  
**And** the user does not have access to job "job_999"  
**When** validating authorization  
**Then** the connection must be rejected with HTTP 403 status  
**And** must return error message "unauthorized access to job: job_999"  
**And** must log the authorization failure attempt

### Requirement: Connection Lifecycle Management MUST Be Implemented
Topic-filtered connections MUST maintain proper lifecycle management including registration, cleanup, and error handling.

#### Scenario: Connection Registration with Topic Filters
**Given** a valid topic-filtered connection is created  
**When** registering the connection with the Hub  
**Then** the Hub must store the connection with its topic filter metadata  
**And** must be able to lookup connections by userID and topic filters  
**And** must maintain indexes for efficient message routing

#### Scenario: Connection Cleanup on Disconnect
**Given** a topic-filtered connection is active  
**When** the connection is closed or disconnected  
**Then** the Hub must remove the connection from all indexes  
**And** must clean up topic filter metadata  
**And** must not leave dangling references that could cause memory leaks

#### Scenario: Connection Error Handling
**Given** a topic-filtered connection encounters an error during message delivery  
**When** the error occurs  
**Then** the connection error must be logged with topic context  
**And** the connection must be cleaned up properly  
**And** other connections for the same user/topic must not be affected

#### Scenario: Graceful Connection Shutdown
**Given** the WebSocket service is shutting down  
**When** graceful shutdown is initiated  
**Then** all topic-filtered connections must be notified  
**And** must be given time to complete in-flight message deliveries  
**And** must be cleanly closed with proper status codes

### Requirement: Connection Limits and Rate Limiting MUST Be Enforced
Topic-based connections MUST enforce appropriate limits to prevent resource exhaustion.

#### Scenario: Project Connection Limits per User
**Given** a user already has 5 project-specific connections  
**When** attempting to create a 6th project connection  
**Then** the connection must be rejected if it exceeds the configured limit  
**And** must return HTTP 429 status with error message "project connection limit exceeded"  
**And** existing connections must remain unaffected

#### Scenario: Job Connection Limits per User  
**Given** a user already has 3 job-specific connections  
**When** attempting to create a 4th job connection  
**Then** the connection must be rejected if it exceeds the configured limit  
**And** must return HTTP 429 status with error message "job connection limit exceeded"  
**And** existing connections must remain unaffected

#### Scenario: General Connection Limits
**Given** configured connection limits are in place  
**When** counting total connections per user  
**Then** general connections must be counted separately from topic-specific connections  
**And** must not prevent topic-specific connections if within their specific limits  
**And** must maintain backward compatibility for general connection limits

#### Scenario: Connection Rate Limiting
**Given** a user is making rapid connection attempts  
**When** the rate exceeds the configured threshold  
**Then** subsequent connection attempts must be rejected with HTTP 429  
**And** must include "Retry-After" header with appropriate delay  
**And** rate limiting must apply per user across all connection types

### Requirement: Connection Monitoring and Metrics SHALL Be Provided
Topic-filtered connections MUST provide comprehensive monitoring and metrics.

#### Scenario: Connection Count Metrics by Topic Type
**Given** topic-filtered connections are active  
**When** collecting metrics  
**Then** connection counts must be tracked by topic type (project, job, general)  
**And** must track connections per user by topic type  
**And** must provide real-time connection statistics

#### Scenario: Connection Lifecycle Metrics
**Given** connections are being created and destroyed  
**When** tracking connection lifecycle  
**Then** metrics must include connection duration by topic type  
**And** must track connection creation/destruction rates  
**And** must monitor connection establishment latency

#### Scenario: Message Delivery Metrics by Topic
**Given** messages are being delivered to topic-filtered connections  
**When** tracking delivery metrics  
**Then** successful delivery count must be tracked by topic type  
**And** failed delivery count must be tracked with failure reasons  
**And** message delivery latency must be measured per topic type

#### Scenario: Authorization Failure Metrics
**Given** authorization checks are occurring  
**When** tracking authorization outcomes  
**Then** successful authorizations must be counted by topic type  
**And** failed authorizations must be counted with failure reasons  
**And** authorization check latency must be measured