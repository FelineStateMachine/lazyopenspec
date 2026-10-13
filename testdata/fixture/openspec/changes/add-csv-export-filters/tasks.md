## 1. API

- [x] 1.1 Parse and validate `status`, `assignee`, `updated_since` query parameters
- [x] 1.2 Parse and validate `columns` parameter against the allowed column set
- [x] 1.3 Apply filters in the export query and stream rows with selected columns
- [x] 1.4 Derive the export file name from the project slug and filters

## 2. Web UI

- [x] 2.1 Add filter and column pickers to the export dialog
- [x] 2.2 Persist last-used export settings per project in local storage

## 3. Verification

- [x] 3.1 Golden-file tests for filtered and column-selected CSV output
- [x] 3.2 Manual check: 50k-task project export stays under 10 MB with filters
