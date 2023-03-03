# watermill-tx-events

## Overview

This prototype shows a very basic architecture to achieve an "outbox" event pattern.

It should be noted that this specific implementation (w/watermill) has a [known bug around
Postgres](https://github.com/ThreeDotsLabs/watermill/issues/311).

## Problem Statement

Messages in GCP PubSub are ephemeral and can go MIA in cases of regional GCP infra
outages or any other major networking incidents (partitions etc.).

## Outbox Pattern

The basic pattern is fairly simple:

**Publisher**

1. In a new db transaction
    2. Create the new record in the business table
    3. Create a new _event_ record to an "outbox" table
    4. If both record creations succeed:
        5. Commit
    6. If either record creations fail:
        7. Rollback

**Subscriber**

1. Poll the outbox table for new records
    2. If new record found, send it to pubsub
    3. Wait for verification that the send was successful

Now in any case, you have a guaranteed event history stored that can be replayed
at any time to recover from any networking issues that may occur and impact your app.
