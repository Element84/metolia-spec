---
title: "Failure code reference"
weight: 140
---

This document consolidates all spec-defined failure codes into a single
reference table. Each code is defined in its home section; this table collects
them for quick lookup. The namespace scheme these codes inhabit is defined with
the failure envelope in
[The Call interface and Result](../call-interface/#code-namespaces).

| Code                                   | Type           | Source               | Defined in                                                                                                     |
| -------------------------------------- | -------------- | -------------------- | -------------------------------------------------------------------------------------------------------------- |
| `System.Cancelled`                     | `cancellation` | Engine               | [Cancellation](../execution-model/#cancellation)                                                               |
| `System.GatherDispatchCancelled`       | `cancellation` | Engine (`Gather`)    | [External cancellation](../execution-model/#external-cancellation)                                             |
| `System.GatherDispatchSkipped`         | `skipped`      | Engine (`Gather`)    | [`System.GatherDispatchSkipped`](../step-actions/#systemgatherdispatchskipped)                                 |
| `System.GatherCompletionUnmet`         | `error`        | Engine (`Gather`)    | [`System.GatherCompletionUnmet`](../step-actions/#systemgathercompletionunmet)                                 |
| `System.ExpressionEvaluationError`     | `error`        | Engine               | [Evaluation errors](../expressions/#evaluation-errors)                                                         |
| `System.ParameterValidationFailed`     | `error`        | Engine               | [`System.ParameterValidationFailed`](../flow-object/#systemparametervalidationfailed)                          |
| `System.UnrepresentableValue`          | `error`        | Engine               | [Evaluation errors](../expressions/#evaluation-errors)                                                         |
| `System.EmptyRaise`                    | `error`        | Engine               | [`System.EmptyRaise`](../step-actions/#systememptyraise)                                                       |
| `System.FailureChainTruncated`         | `error`        | Engine               | [Chaining](../execution-context/#chaining)                                                                     |
| `Provider.Middleware.Retry.Exhausted`  | `error`        | `Retry` middleware   | [`Provider.Middleware.Retry.Exhausted`](../providers/middleware-providers/#providermiddlewareretryexhausted)   |
| `Provider.Middleware.Timeout.Exceeded` | `timeout`      | `Timeout` middleware | [`Provider.Middleware.Timeout.Exceeded`](../providers/middleware-providers/#providermiddlewaretimeoutexceeded) |
