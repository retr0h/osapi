# Proposal: Transitioning API Request Handling to NATS with KV Store

**Date:** March 2025
**Status:** Draft
**Author:** @retr0h

## Introduction

This Proposal outlines the transition of **GET request handling** from a direct
execution model to a **fully NATS-driven architecture**, leveraging **JetStream
KV stores** for response persistence. The **POST request flow remains
unchanged**, relying on **protobuf-encoded messages in NATS** for async
processing.

Additionally, this Proposal introduces a **unified job worker**, responsible
for processing both **write operations (modifications)** and **read operations
(queries)**. The subject hierarchy will be structured to distinguish between
these two types of operations.

## Current Architecture

Currently, **GET and POST requests are handled differently**, as shown below:

### Key Behaviors:

1. **GET (`read`) Requests**:
   - Bypasses NATS
   - API directly invokes the required **provider service**
   - Response is returned synchronously to the client

2. **POST (`write`) Requests**:
   - Encodes a **protobuf message** into NATS
   - A **worker** picks up the job asynchronously
   - The worker performs the requested operation and **acknowledges** completion

## Proposed Architecture

We propose moving **GET requests fully into NATS**, using **JetStream KV stores**
for response persistence.

Additionally, a **single job worker** will now process both:
- **Write subjects**: Requests that modify state
- **Read subjects**: Requests that query state and return results via KV

### **Message Flow Using `REQUEST_ID`**

To ensure **traceability and deduplication**, all API requests will carry a
**unique `REQUEST_ID`**, which will be stored in the **JetStream KV store** for
retrieval. The process is as follows:

1. **API publishes a request to NATS**:
   - A unique `REQUEST_ID` is generated
   - The request is sent to the correct **query or modify** subject
   - The **`REQUEST_ID` is included both in the payload and as a NATS header (`NATS-Msg-Id`)**

2. Worker processes the request:
   - Extracts the REQUEST_ID from the JSON body
   - Executes the necessary operation
   - Stores the response in the KV store under the REQUEST_ID

3. API retrieves the response asynchronously:
   - Instead of waiting for a direct response, the API polls the KV store for the stored response

### **Subject Breakdown**

To distinguish between **write (change)** and **read (query)** operations, we
propose the following subject hierarchy:

- **Write Requests**:
  - `jobs.modify.dns`
  - `jobs.modify.disk`
  - `jobs.modify.*`

- **Read Requests**:
  - `jobs.query.dns`
  - `jobs.query.disk`
  - `jobs.query.*`

### **Key Changes:**

**GET (`read`) Requests:**
- Now traverse **NATS JetStream**
- The **worker** processes the request **asynchronously**
- The **response is stored in a JetStream KV bucket**
- The **Async Response Client** retrieves responses later

**POST (`write`) Requests:**
- **No changes** from the current implementation
- Workers still listen for job requests and execute tasks asynchronously

**Unified Job Worker:**
- **Listens on both read (`jobs.query.*`) and write (`jobs.modify.*`) subjects**
- **Processes state-changing and query operations separately**
- **Stores read responses in KV for later retrieval by the API**

## Benefits of the New Model

### Enhanced Decoupling

- API **no longer directly invokes the provider**
- The **worker independently processes requests**, allowing for **horizontal scaling**

### Persistent Responses

- API clients can **retrieve responses asynchronously**
- **Improves fault tolerance**—clients can fetch responses even if they weren’t actively waiting

### Clear Separation of Read & Write Jobs

- **Write (`jobs.modify.*`) and Read (`jobs.query.*`) jobs are logically separate**
- This prevents **unnecessary state modifications** when only a query is needed

### Unified Request Handling

- All API requests (GET & POST) now traverse **NATS**
- Provides a **consistent event-driven architecture**
