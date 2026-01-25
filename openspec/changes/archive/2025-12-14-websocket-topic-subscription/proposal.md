# WebSocket Topic-Based Subscription Implementation

**Change ID**: `websocket-topic-subscription`  
**Type**: Feature Enhancement  
**Priority**: High  
**Status**: Draft  
**Created**: 2025-12-13

## Problem Statement

The current WebSocket notification service only supports user-based subscriptions with pattern `user_noti:*`, where all messages for a user are delivered to all their connections. This creates several issues:

1. **Message Noise**: Clients receive irrelevant notifications
2. **Bandwidth Waste**: Unnecessary message transmission 
3. **Poor User Experience**: Users get spammed with unrelated notifications
4. **Lack of Context**: Messages lack specific project/job targeting

## Proposed Solution

Implement **Topic-Based Subscription** system with two main subscription types:

1. **Project Topic**: `project:projectID:userID` - Project-specific notifications (progress, completion, errors)
2. **Job Topic**: `job:jobID:userID` - Job-specific notifications (batch results, crawl progress)

### Key Features

- **Targeted Message Delivery**: Clients only receive relevant messages for their subscribed topics
- **Mandatory Transform Layer**: WebSocket service transforms Redis input format to standardized output format 
- **Multi-Pattern Subscription**: Redis subscribes to `project:*` and `job:*` patterns
- **Authorization Control**: Validate user access to specific projects/jobs before connection
- **Backward Compatibility**: Existing clients continue to work without changes
- **Publisher Compatibility**: Publishers keep existing message format

## Business Impact

- **Performance**: 60-80% reduction in unnecessary message transmission
- **User Experience**: Contextual notifications improve engagement
- **Scalability**: More efficient resource utilization
- **Maintainability**: Clean separation between publishers and clients through transform layer

## Technical Approach

### 1. WebSocket Service Enhancements
- Add topic-based connection filtering
- Implement mandatory transform layer for message processing
- Extend Hub pattern with topic-aware message routing
- Add connection parameter parsing for projectId/jobId

### 2. Message Transformation
- Transform Redis input messages to standardized output format
- Validate and normalize message structures
- Handle omitempty fields correctly
- Centralized error handling for malformed messages

### 3. Security & Authorization
- Validate user access to projects/jobs before WebSocket connection
- Input validation for projectId/jobId parameters
- Rate limiting for topic-specific connections

## Success Criteria

1. **Functional**: 
   - Topic-based message filtering works correctly
   - Transform layer processes all message types
   - Authorization prevents unauthorized access

2. **Performance**:
   - <30% memory increase per connection
   - <5ms message filtering latency 
   - 60-80% reduction in unnecessary messages

3. **Quality**:
   - 100% backward compatibility
   - Zero message loss during transformation
   - Graceful error handling for malformed messages

## Dependencies

- Redis Pub/Sub infrastructure (existing)
- Authentication service for project/job access validation
- Existing publisher services (no changes required)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Transform layer failures | High | Comprehensive error handling, fallback mechanisms |
| Memory overhead | Medium | Connection limits, efficient data structures |
| Redis pattern subscription load | Medium | Performance monitoring, optimization |
| Breaking existing clients | High | Backward compatibility testing |

## Next Steps

1. Create detailed technical specifications for each capability
2. Design transform layer architecture and validation rules
3. Plan phased implementation with feature flags
4. Develop comprehensive testing strategy