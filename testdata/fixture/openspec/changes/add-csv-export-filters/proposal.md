## Why

Project owners export to CSV mainly to build weekly status reports, and every one of them immediately deletes most rows and columns in a spreadsheet. Letting them filter the export up front saves that step and keeps large exports under the 10 MB email attachment limit.

## What Changes

- CSV export accepts optional `status`, `assignee`, and `updated_since` filters
- CSV export accepts a `columns` parameter to choose and order columns
- Filters are echoed in the export file name so reports are self-describing

## Capabilities

### New Capabilities
<!-- None -->

### Modified Capabilities
- `data-export`: CSV export gains column selection; new export filters requirement

## Impact

- `GET /api/projects/:id/export.csv` query parameters (backwards compatible)
- Export dialog in the web UI gains filter controls
