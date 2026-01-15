# Frontend Rework Design: golang-dashboard UI/UX Pattern

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the existing Tailwind-based frontend with MUI (Material UI) using golang-dashboard's neobrutalism design system.

**Architecture:** Keep Go backend (managerrunner) serving API at `/api/v2/*`. Replace `web/frontend/` React app with MUI components matching golang-dashboard visual patterns.

**Tech Stack:** React 19 + MUI v7 + Recharts + TanStack Query + TypeScript

---

## Design Overview

### Visual Design System (golang-dashboard)

The golang-dashboard uses a "neobrutalism" design:
- **Thick black borders**: 2px solid #000000 on cards/buttons
- **Yellow accent**: #FFD93D for primary actions and highlights
- **Clean typography**: Inter font, bold headings
- **Status colors**:
  - Success: #22C55E (green)
  - Error: #EF4444 (red)
  - Warning: #F59E0B (amber)
  - Info: #3B82F6 (blue)
- **Border radius**: 16px for cards, 8px for buttons
- **No shadows**: boxShadow: 'none' everywhere

### Dashboard Layout

```
+------------------------------------------------------------------+
|  Header: "Dashboard" + Subtitle + Refresh Button                  |
+------------------------------------------------------------------+
|  [Stat] [Stat] [Stat] [Stat]   <- 4 columns (or 2x2 on mobile)   |
|  [Stat] [Stat] [Stat] [Stat]   <- 8 total stat cards             |
+------------------------------------------------------------------+
|  [Hourly Activity Chart - 70%]  |  [Recent Activity - 30%]       |
|  - Bar chart with Recharts      |  - List of recent scrapes      |
+------------------------------------------------------------------+
|  [Worker Status - 40%]          |  [Job Progress Table - 60%]    |
|  - Color-coded worker cards     |  - Progress bars per job       |
+------------------------------------------------------------------+
```

### Stat Cards (8 cards, 4x2 grid)

1. **Total Jobs** - All jobs count with pending subtitle
2. **Active Jobs** - Currently running with "processing" subtitle
3. **Completed** - Success count with success rate %
4. **Failed** - Failed count with retry pending count
5. **Total Results** - Business listings scraped
6. **Emails Found** - With valid email count
7. **Online Workers** - With total worker count
8. **Rate** - Results/hour throughput

### Pages to Update

| Page | Current | Target |
|------|---------|--------|
| Dashboard | 4 stat cards, simple layout | 8 stat cards, charts, activity feed |
| Jobs | Basic table | MUI DataGrid with filters, status chips |
| Job Detail | Basic info | Rich detail with progress, results table |
| Workers | Simple list | Status cards with health indicators |
| Results | Basic table | Advanced filtering, column selection |
| Settings | Basic form | MUI form components |

---

## Component Mapping

### Current -> Target

| Current (Tailwind) | Target (MUI) |
|--------------------|--------------|
| `Card` (custom) | `MUI Card` with custom theme |
| `Badge` (custom) | `MUI Chip` with status colors |
| `Button` (custom) | `MUI Button` with yellow accent |
| `Table` (custom) | `MUI Table` or `DataGrid` |
| `Input` (custom) | `MUI TextField` |
| lucide-react icons | @mui/icons-material |

### New Components

- `StatCard` - Reusable stat card with icon, value, subtitle
- `StatusChip` - Color-coded status indicator
- `ActivityFeed` - Recent activity list with timestamps
- `HourlyChart` - Recharts bar chart for hourly stats
- `ProgressBar` - Job progress indicator

---

## Files to Modify

### Package Changes
```
Remove:
- tailwindcss, postcss, autoprefixer
- class-variance-authority, clsx, tailwind-merge

Add:
- @mui/material @mui/icons-material @emotion/react @emotion/styled
```

### File Changes

**Replace:**
- `src/App.tsx` - Add MUI ThemeProvider
- `src/pages/Dashboard.tsx` - Full redesign
- `src/pages/Jobs.tsx` - MUI table
- `src/pages/Jobs/JobDetail.tsx` - Rich detail view
- `src/pages/Workers.tsx` - Status cards
- `src/pages/Results.tsx` - Advanced table
- `src/components/Layout/AppShell.tsx` - MUI layout

**Delete:**
- `src/components/UI/Card.tsx` (use MUI)
- `src/components/UI/Badge.tsx` (use MUI Chip)
- `src/components/UI/Button.tsx` (use MUI)
- `src/components/UI/Input.tsx` (use MUI TextField)
- `tailwind.config.js`
- `postcss.config.js`

**Create:**
- `src/theme.ts` - MUI theme matching golang-dashboard
- `src/components/StatCard.tsx` - Reusable stat card
- `src/components/StatusChip.tsx` - Status indicator
- `src/components/ActivityFeed.tsx` - Recent activity
- `src/components/HourlyChart.tsx` - Bar chart

---

## API Compatibility

The Go backend API at `/api/v2/*` remains unchanged. Current endpoints:

| Endpoint | Used For |
|----------|----------|
| `GET /api/v2/stats` | Dashboard stats |
| `GET /api/v2/jobs` | Jobs list |
| `GET /api/v2/jobs/:id` | Job detail |
| `GET /api/v2/jobs/:id/results` | Job results |
| `GET /api/v2/workers` | Workers list |
| `GET /api/v2/results` | All results |
| `POST /api/v2/jobs` | Create job |

### Missing Endpoints (may need to add)

- `GET /api/v2/stats/hourly` - For hourly activity chart
- `GET /api/v2/activity/recent` - For recent activity feed

---

## Implementation Priority

1. **Theme + Foundation** - Create MUI theme, update App.tsx
2. **Dashboard** - Most visible, highest impact
3. **Jobs** - Core functionality
4. **Workers** - Status monitoring
5. **Results** - Data browsing
6. **Settings** - Lower priority

---

## Verification Checklist

- [ ] MUI theme matches golang-dashboard colors
- [ ] Dashboard shows 8 stat cards in 2x4 grid
- [ ] Hourly chart displays with Recharts
- [ ] Recent activity feed works
- [ ] Worker status cards show correct colors
- [ ] Job progress bars functional
- [ ] Mobile responsive (2 columns on mobile)
- [ ] All existing functionality preserved
- [ ] Build passes (`npm run build`)
- [ ] No console errors
