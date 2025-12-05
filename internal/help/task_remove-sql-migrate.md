# 2025.12.02

Vasily Tsybenko
> I would say we a have a big tech-debt in https://github.com/acronis/go-dbkit/tree/main/migrate. We use github.com/rubenv/sql-migrate 
under the hood and there are multiple issues with this (like https://github.com/rubenv/sql-migrate/issues/211). My idea here is to 
get rid github.com/rubenv/sql-migrate off with all transitive dependencies and reimplement the same minimal enough functionality right 
in go-dbkit. But before implementing, I think it's worth to brainstorm that idea with AI. Sonnet 4.5 is a good model for this.

## **Initial Plan: go-dbkit/migrate Refactoring (Replacing sql-migrate)**
by ChatGPT

### **1. Problem Summary**

* `github.com/rubenv/sql-migrate` is used internally in `go-dbkit/migrate`.
* The library brings **significant tech-debt and transitive dependencies**.
* Multiple issues exist (e.g. sql-migrate#211) that impact maintainability and reliability.
* The long-term goal is to **remove sql-migrate** and implement the minimal required functionality directly in go-dbkit.

---

## **2. Objectives**

1. Reduce external dependencies and transitive bloat.
2. Improve maintainability, transparency, and testing.
3. Preserve *100% backward compatibility* with the current migrate package.
4. Ensure the new solution covers only what go-dbkit truly needs — not the full sql-migrate feature set.

---

## **3. Short-Term Steps**

### **(A) Requirements Gathering**

* Identify exactly which sql-migrate features are used by go-dbkit:

  * Migration sources (embedded, filesystem, go code)
  * Up/Down operations
  * Transaction semantics
  * Dialect-specific behavior
  * Migration table format
* List incompatible/buggy behaviors today (including known issues like #211).

### **(B) Analyze Current Architecture**

* Document how go-dbkit wraps sql-migrate today.
* Identify touch points → where replacing the engine would affect interfaces.

---

## **4. Brainstorming With AI (Sonnet 4.5 or similar)**

Topics to explore with AI:

* Minimal viable migration engine architecture.
* Tradeoffs: stateful vs stateless migration logic.
* Migration table schema design.
* Best design for adapters per SQL dialect.
* How to preserve backward compatibility with existing migration files.
* Potential simplifications compared to sql-migrate.

Goal: produce a **draft design doc** before coding.

---

## **5. Mid-Term Steps**

### **(A) Create a Prototype Migration Engine**

* Focus on:

  * Reading migration sets
  * Sorting & applying them deterministically
  * Managing the migration history table
  * Basic dialect abstraction (pgx, mssql, mysql, sqlite)

### **(B) Test Parity**

* Use existing go-dbkit migration tests + new tests to ensure identical behavior.
* Build targeted regression tests for known sql-migrate issues.

### **(C) Compatibility Verification**

* Validate that:

  * Existing migration files parse correctly
  * Existing applications using go-dbkit/migrate behave unchanged

---

## **6. Final Steps**

* Replace sql-migrate internally while keeping public API stable.
* Remove transitive dependencies.
* Add documentation + migration notes (for developers, not users).
* Perform internal review and finalize integration into main.

---

If you want, I can also draft:
✔ a design document template
✔ a detailed AI prompt for Sonnet brainstorming
✔ an estimated timeline and required effort


