# Topic-Based Message Filtering

## ADDED Requirements

### Requirement: Topic Pattern Subscription SHALL Be Implemented
The WebSocket service MUST subscribe to multiple Redis topic patterns to receive different types of notifications.

#### Scenario: Redis Pattern Subscription Setup
**Given** the WebSocket service is starting up  
**When** initializing the Redis subscriber  
**Then** it must subscribe to the patterns `project:*` and `job:*`  
**And** maintain existing functionality for any other patterns currently used

#### Scenario: Multi-Pattern Message Handling  
**Given** the service is subscribed to multiple Redis patterns  
**When** a message arrives on channel `project:proj_123:user_456`  
**Then** the channel must be parsed to extract `projectID=proj_123` and `userID=user_456`  
**And** the message must be routed to the project notification handler  

#### Scenario: Invalid Channel Format Handling
**Given** a message arrives on an invalid channel format  
**When** the channel cannot be parsed into the expected pattern  
**Then** the service must log an error with the invalid channel name  
**And** must not crash or affect other message processing  

### Requirement: Connection Topic Filtering MUST Be Supported
WebSocket connections MUST be able to subscribe to specific topics based on query parameters.

#### Scenario: Project-Specific Connection
**Given** a user connects with URL `ws://localhost:8081/ws?projectId=proj_123`  
**When** the connection is established  
**Then** the connection must only receive messages published to `project:proj_123:userID`  
**And** must not receive messages from other projects or jobs  

#### Scenario: Job-Specific Connection  
**Given** a user connects with URL `ws://localhost:8081/ws?jobId=job_789`  
**When** the connection is established  
**Then** the connection must only receive messages published to `job:job_789:userID`  
**And** must not receive messages from other jobs or projects

#### Scenario: General Connection Backward Compatibility
**Given** a user connects with URL `ws://localhost:8081/ws` (no topic parameters)  
**When** the connection is established  
**Then** the connection must receive all messages for that user (existing behavior)  
**And** must maintain full backward compatibility with existing clients

#### Scenario: Multiple Topic Connections
**Given** a user has multiple WebSocket connections with different topic filters  
**When** a message is published to `project:proj_123:userID`  
**Then** only connections with `projectId=proj_123` must receive the message  
**And** connections with different or no filters must not receive the message

### Requirement: Hub Message Routing Enhancement SHALL Be Implemented
The WebSocket Hub MUST route messages based on topic subscription filters.

#### Scenario: Project Message Routing
**Given** a project notification message arrives for `project:proj_123:user_456`  
**When** the hub processes the message  
**Then** it must identify all connections for `user_456` with `projectID=proj_123`  
**And** must send the message only to those filtered connections

#### Scenario: Job Message Routing
**Given** a job notification message arrives for `job:job_789:user_456`  
**When** the hub processes the message  
**Then** it must identify all connections for `user_456` with `jobID=job_789`  
**And** must send the message only to those filtered connections  

#### Scenario: No Matching Connections
**Given** a message arrives for a topic with no active connections  
**When** the hub attempts to route the message  
**Then** it must log the delivery attempt  
**And** must not error or affect other message processing  
**And** must update metrics to track undelivered messages

### Requirement: Topic Parameter Validation MUST Be Enforced
All topic parameters MUST be validated for security and format compliance.

#### Scenario: Valid Project ID Format
**Given** a connection request with `projectId=proj_123_test`  
**When** validating the project ID  
**Then** it must accept alphanumeric characters, underscores, and hyphens  
**And** must accept length between 1-50 characters

#### Scenario: Invalid Project ID Format  
**Given** a connection request with `projectId=proj@123#`  
**When** validating the project ID  
**Then** it must reject the connection with HTTP 400  
**And** must return error message "invalid project ID format"

#### Scenario: Empty Topic Parameters
**Given** a connection request with `projectId=` (empty value)  
**When** validating the parameters  
**Then** it must treat the empty parameter as if not provided  
**And** must establish a general connection without filtering