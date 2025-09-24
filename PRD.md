📄 Product Requirements Document (PRD)

Product Name: CMDB Lite
Version: 1.1
Date: 2025-09-20
Author: [Your Name]

1. 🎯 Purpose

CMDB Lite is a lightweight, modern Configuration Management Database (CMDB) that helps IT teams store, track, and visualize Configuration Items (CIs) and their relationships with minimal complexity.
It is designed for small to mid-sized organizations or teams that don’t need a full ITSM suite but want visibility and auditability over their infrastructure.

2. 📌 Goals & Objectives

✅ Provide a simple, intuitive UI for managing CIs

✅ Allow users to define relationships (e.g., server → application → database)

✅ Offer graph visualization to understand dependencies

✅ Track changes with full audit logs

✅ Be easy to deploy (Docker/Kubernetes) and integrate into DevOps workflows

3. 🎯 Target Users

IT Operations – manage infrastructure components

DevOps / SRE – visualize service dependencies

Security / Audit Teams – trace changes and incidents

4. 📦 Product Scope
4.1 In Scope

CI CRUD (Create, Read, Update, Delete)

Relationship management

CI visualization (force-directed graph)

Audit logging for all changes

Authentication & role-based access control (RBAC)

REST API for automation/integration

Deployment: Docker Compose + Kubernetes support

4.2 Out of Scope

ITIL Process Modules (Incident, Change, Problem)

Workflow engine (manual relationships only)

CMDB discovery agents (manual entry only for v1)

5. 🖼️ High-Level User Flow
flowchart TD
    A[Login] --> B[Dashboard]
    B --> C[View CI List]
    B --> D[Create CI]
    B --> E[Search CI]
    C --> F[View CI Details]
    F --> G[Edit CI]
    F --> H[Delete CI]
    F --> I[View Relationships Graph]
    I --> J[Add Relationship]
    I --> K[Remove Relationship]
    B --> L[View Audit Logs]

6. 🧩 Key Features
Feature	Description
CI Management	Users can add servers, apps, DBs, licenses, etc., with attributes (name, type, tags).
Relationship Graph	Visualize dependencies between CIs with interactive graph (click to expand/collapse).
Search & Filter	Search by name, type, or tags; filter by CI type.
Audit Logs	Track changes (who, when, what).
Auth & RBAC	Login system with admin/viewer roles.
REST API	Programmatic access for CI and relationship management.
Deployment Simplicity	Single docker-compose.yml, optional Helm chart.
7. 🎨 UI/UX Mockup (Concept)
flowchart LR
    subgraph UI
        A[Top Nav] --> B[Sidebar: CI Types]
        B --> C[CI List View]
        C --> D[CI Detail Drawer]
        D --> E[Graph Visualization]
    end


Top Nav: Search bar + user profile

Sidebar: Filters by CI type

Main Area: List or graph view

Detail Drawer: Appears when selecting a CI

8. ✅ Success Metrics

🟢 Ease of Adoption: Deployed in <10 min (docker-compose)

🟢 User Engagement: 80%+ users create >5 CIs within first session

🟢 Performance: API responds <200ms for CI CRUD on <10k records

🟢 Reliability: No data loss during container restarts

9. 📅 Release Plan

MVP (v1.0): CI CRUD, relationships, visualization, audit logs, auth

Future Enhancements: CI import/export (CSV/JSON), integration with CM tools (Ansible, Terraform)

10. 🚫 Risks & Assumptions
Risk	Mitigation
Over-engineering (too complex for small teams)	Keep scope tight, focus on simplicity
Performance issues with >50k CIs	Optimize queries, add pagination, caching
Security vulnerabilities	Implement proper JWT expiration & hashing, run security scans