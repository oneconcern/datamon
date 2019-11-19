# Context

## Definition

A context is a way to define multiple instances of Datamon. 
It consists of a unique set of storage buckets with a specific set of storage policies that can be enforced
using features provided by cloud providers.

1. Blob bucket: stores the raw storage blobs for CAFS. These blobs are write once/read many. 
2. Metadata bucket: Metadata bucket stores all the write once, never updated metadata.
3. Versioned Metadata: Versioned metadata bucket stores metadata that can be updated. Example labels.
The objects in this bucket are versioned and the history of the updates can be enumerated.
4. WAL: WAL buckets store the Write Ahead Log.
5. Read Log: logs all read operations performed.

## Implementing Development and Production data sets with data sharing

Context allows for implementing operational concepts such as production datasets and allow for 
gate-keeping when managing datasets.

The Blob bucket can be shared between development and production and different access controls can 
be implemented for development and production.

As an example, a Git + CI system can be implemented that reviews the scripts and datasets that need
to be moved from development to production. The repo/bundle can be moved from development to production
without creating multiple copies of the data.

Development context can be configured with more liberal access control policies, whereas production 
can be locked down to only the gatekeeping mechanism as the source of change.

## Configuration

The CLI supports the ability to host a configuration bucket that hosts all the contexts and enforce
the selection of buckets to form a Context.

