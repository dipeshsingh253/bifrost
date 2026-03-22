# 🧠 Product Requirements Document (PRD)

## 1. Overview

**Product Name:** VPS Monitoring Dashboard  
**Owner:** Shivay  
**Created Date:** 2026-03-22  
**Last Updated:** 2026-03-22  
**Version:** v1.0

### Summary

A monitoring tool for VPS servers that works in two modes: **self-hosted (free)** and **cloud-hosted SaaS (paid)**.  
It provides server metrics, Docker visibility, and logs in one unified dashboard.

Goal:  
Replace complex setups like Grafana + Loki with a simple, fast, and focused solution.

---

### Deployment Model

**1. Self-Hosted**

- User runs agent + dashboard
    
- Free forever
    
- Limited features
    

**2. Cloud (SaaS)**

- User runs only the agent
    
- Dashboard is hosted
    
- Includes alerts, retention, and team features
    

---

## 2. Problem Statement

### Problem

Monitoring small VPS setups requires multiple tools and manual effort.

### Who is affected?

- Solo developers
    
- Small teams (2–40 people)
    

### Why is it important?

- Debugging is slow
    
- Tooling is overly complex
    
- Visibility is fragmented
    

### Pain Points

- Metrics and logs are separate
    
- Complex setup (Grafana/Loki)
    
- Frequent SSH usage
    
- Poor service-level visibility
    

---

## 3. Goals & Success Metrics

### Objectives

- Provide one unified monitoring dashboard
    
- Support both self-hosted and cloud usage
    
- Enable fast debugging without SSH
    
- Keep setup extremely simple
    

### Success Metrics (KPIs)

- Setup time < 10 minutes
    
- Issue identification time < 30 seconds
    
- Users rely on a single dashboard
    
- High adoption of free → cloud upgrade
    

---

## 4. Target Users

### Primary Users

- Solo developers
    
- Small teams (2–40 people)
    

### Secondary Users

- Teams without dedicated DevOps
    

### User Personas (Optional)

**Persona 1:**  
Indie developer managing a few VPS servers

**Persona 2:**  
Startup team running multiple Docker services

---

## 5. Scope

### In Scope (MVP)

- Server metrics (CPU, RAM, disk, network)
    
- Docker/container monitoring
    
- Docker-compose grouping
    
- Container logs (tail + search)
    
- Lightweight agent
    
- RBAC (basic)
    
- Self-hosted dashboard
    
- Cloud-hosted dashboard (multi-tenant)
    
- Tier-based features (retention, alerts)
    

### Out of Scope

- Kubernetes support
    
- Distributed tracing
    
- Advanced analytics
    
- Complex dashboards
    

---

## 6. Features & Requirements

### Core Features

---

### Feature 1: Server Overview

**Description:**  
View all servers with key metrics.

**Why it matters:**  
Quick system health visibility.

**Acceptance Criteria:**

- Shows CPU, memory, disk, network
    
- Supports multiple servers
    
- Auto-refresh
    

---

### Feature 2: Server Detail View

**Description:**  
Detailed server-level monitoring.

**Why it matters:**  
Diagnose host issues quickly.

**Acceptance Criteria:**

- Host metrics breakdown
    
- OS info and uptime
    
- Firewall status and open ports
    
- Docker services list
    

---

### Feature 3: Service / Stack View

**Description:**  
Group containers by service or compose project.

**Acceptance Criteria:**

- Grouped container view
    
- Shows status, uptime, restarts
    
- Displays ports and health
    
- Supports standalone containers
    

---

### Feature 4: Container Detail View

**Description:**  
Detailed container-level monitoring.

**Acceptance Criteria:**

- CPU, memory, IO metrics
    
- Restart count
    
- Uptime
    
- Status
    

---

### Feature 5: Logs

**Description:**  
View container logs in UI.

**Acceptance Criteria:**

- Live tail
    
- Scroll history
    
- Keyword search
    
- Basic time filtering
    

---

### Feature 6: RBAC

**Description:**  
Control access for teams.

**Acceptance Criteria:**

- Roles: Owner, Viewer
    
- Server-level permissions
    
- Basic service-level visibility
    

---

### Feature 7: Agent

**Description:**  
Lightweight service on each VPS.

**Acceptance Criteria:**

- Runs via Docker or systemd
    
- Collects host + Docker metrics
    
- Streams logs
    
- YAML configuration
    
- Simple install
    

---

## 7. User Flows

### Example Flow

1. User opens dashboard
    
2. Selects a server
    
3. Views services
    
4. Opens a container
    
5. Checks logs
    
6. Identifies issue
    

---

## 8. Technical Specification

### Tech Stack

**Frontend:**

- Next.js (React + TypeScript)
    

**Backend:**

- Go + Gin
    

**Database:**

- PostgreSQL
    

**Infrastructure:**

- Docker-based
    
- Self-hosted or cloud deployment
    

---

### Architecture Notes

- Agent runs on each VPS
    
- Agent sends data to:
    
    - Self-hosted backend OR
        
    - Cloud backend (multi-tenant)
        
- Backend handles:
    
    - Data ingestion
        
    - Short-term storage
        
    - RBAC
        
- Frontend connects to backend
    
- Cloud version includes:
    
    - Multi-tenant isolation
        
    - Usage tracking
        

---

### Data Models (Optional)

```json
{
  "server": {
    "id": "string",
    "name": "string"
  },
  "container": {
    "id": "string",
    "name": "string",
    "status": "running"
  }
}
```

---

## 9. Constraints & Assumptions

### Constraints

**Time:**

- Fast MVP delivery (UI/UX is very important here)
    

**Resources:**

- 1–2 developers
    

**Technical:**

- Docker-based environments
    

---

### Assumptions

- Users use Docker/docker-compose
    
- Simplicity is preferred over flexibility
    
- Short log retention is acceptable
    

---

## 10. Risks & Edge Cases

### Risks

- Log volume growth
    
- Performance under multiple servers
    
- Docker API inconsistencies
    

### Edge Cases

- Rapid container restarts
    
- Large logs
    
- Agent disconnects
    
- Partial data
    

---

## 11. Future Improvements (Post-MVP)

- Uptime monitoring
    
- Alerts (Slack/email)
    
- Long-term log storage
    
- Advanced RBAC
    
- API access
    

---

## 12. Pricing

### Free (Self-Hosted)

**Price:** Free forever

Includes:

- Unlimited servers
    
- Unlimited services and containers
    
- Basic monitoring
    
- Logs (24h retention)
    
- Basic RBAC
    

Limitations:

- No hosted dashboard
    
- No alerts
    
- No long-term storage
    

---

### Cloud Starter

**Price:** $10 / server / month  
**Limit:** Up to 3 servers

Includes:

- Hosted dashboard
    
- Real-time monitoring
    
- 7-day logs
    
- Basic alerts
    
- Simple RBAC
    

---

### Cloud Pro

**Price:** $15 / server / month

Includes:

- 30-day logs
    
- Full alerting
    
- Advanced RBAC
    
- Service-level insights
    

---

### Enterprise

**Price:** Custom

Includes:

- SSO
    
- Audit logs
    
- Custom retention
    
- SLA support
    
- Private deployments
    

---

### Pricing Principles

- Pricing based on servers only
    
- No per-log or per-metric pricing
    
- Unlimited services and containers
    
- Predictable billing
    

