# UI Generation System

The AI SDK includes a comprehensive UI generation system that allows AI agents to create rich, interactive user interfaces programmatically. This system enables agents to output structured, typed content parts that frontend applications can render as sophisticated UI components.

## Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [ContentPart Types](#contentpart-types)
- [UI Components](#ui-components)
- [Presentation Tools](#presentation-tools)
- [Usage Examples](#usage-examples)
- [Integration Guide](#integration-guide)
- [Best Practices](#best-practices)

---

## Overview

### What is UI Generation?

Instead of returning plain text, AI agents can generate **structured responses** containing rich UI components like:
- Tables, charts, and metrics dashboards
- Interactive forms and buttons
- Timeline and kanban boards
- Cards, alerts, and progress indicators
- Galleries, carousels, and more

### Key Benefits

✅ **Type-Safe**: All components use Go structs with full type safety  
✅ **Declarative**: Describe what to render, not how to render it  
✅ **Framework-Agnostic**: JSON output works with any frontend (React, Vue, Angular, etc.)  
✅ **AI-Native**: Presentation tools designed for LLM tool calling  
✅ **Streaming**: Real-time progressive rendering of complex UIs  
✅ **Composable**: Combine multiple content parts into rich responses  

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    AI Agent (LLM)                            │
│  Decides what UI to generate based on task                  │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Presentation Tools                              │
│  12 built-in tools AI can call:                            │
│  - render_table, render_chart, render_metrics              │
│  - render_timeline, render_kanban, render_buttons          │
│  - render_form, render_card, render_stats, etc.            │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              StructuredResponse                              │
│  ┌─────────────────────────────────────────────────┐       │
│  │ ContentPart #1: Text                             │       │
│  │ ContentPart #2: Table                            │       │
│  │ ContentPart #3: Chart                            │       │
│  │ ContentPart #4: ButtonGroup                      │       │
│  └─────────────────────────────────────────────────┘       │
│  + Artifacts, Citations, Suggestions, Metadata              │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼ JSON
┌─────────────────────────────────────────────────────────────┐
│              Frontend Application                            │
│  Renders ContentParts as native UI components               │
│  (React, Vue, Angular, Svelte, etc.)                        │
└─────────────────────────────────────────────────────────────┘
```

---

## ContentPart Types

All content parts implement the `ContentPart` interface:

```go
type ContentPart interface {
    Type() ContentPartType
    ToJSON() ([]byte, error)
}
```

### Base ContentPartTypes

| Type | Purpose | Use Case |
|------|---------|----------|
| `PartTypeText` | Plain or formatted text | Paragraphs, explanations |
| `PartTypeMarkdown` | Markdown content | Rich formatted text |
| `PartTypeCode` | Syntax-highlighted code | Code snippets, examples |
| `PartTypeTable` | Tabular data | Data grids, comparisons |
| `PartTypeCard` | Contained content | Highlights, summaries |
| `PartTypeList` | Ordered/unordered lists | Steps, options, items |
| `PartTypeImage` | Images with captions | Visual content |
| `PartTypeThinking` | AI reasoning process | Show thought process |
| `PartTypeQuote` | Quotations | Citations, references |
| `PartTypeDivider` | Visual separator | Section breaks |
| `PartTypeAlert` | Notifications | Warnings, errors, success |
| `PartTypeProgress` | Progress indicators | Loading, completion |
| `PartTypeChart` | Data visualization | Trends, distributions |
| `PartTypeJSON` | Structured JSON | API responses, config |
| `PartTypeCollapsible` | Expandable content | Long content, details |

### Advanced UI ContentPartTypes

| Type | Purpose | Use Case |
|------|---------|----------|
| `PartTypeButtonGroup` | Interactive buttons | Action choices, navigation |
| `PartTypeTimeline` | Chronological events | History, milestones |
| `PartTypeKanban` | Board with cards | Task management, workflows |
| `PartTypeMetric` | KPIs with trends | Dashboards, analytics |
| `PartTypeForm` | Input collection | Data entry, filters |
| `PartTypeTabs` | Tabbed content | Organized content |
| `PartTypeAccordion` | Collapsible sections | FAQs, documentation |
| `PartTypeInlineCitation` | Inline source refs | Academic, research |
| `PartTypeStats` | Compact statistics | Quick overviews |
| `PartTypeCarousel` | Sliding content | Image galleries, features |
| `PartTypeGallery` | Media collection | Photos, portfolios |

---

## UI Components

### 1. ButtonGroup - Interactive Actions

```go
type ButtonGroupPart struct {
    Title       string
    Description string
    Buttons     []Button
    Layout      ButtonLayout // horizontal, vertical, grid, wrap
    Alignment   string       // left, center, right
}

type Button struct {
    ID       string
    Label    string
    Icon     string
    Action   ButtonAction
    Variant  ButtonVariant // primary, secondary, outline, ghost, danger
    Size     string        // sm, md, lg
    Disabled bool
    Tooltip  string
}
```

**Example:**
```go
buttons := NewButtonGroupPart(
    Button{
        ID:      "approve",
        Label:   "Approve",
        Variant: ButtonPrimary,
        Action:  ButtonAction{Type: ActionTypeTool, Value: "approve_request"},
    },
    Button{
        ID:      "reject",
        Label:   "Reject",
        Variant: ButtonDanger,
        Action:  ButtonAction{Type: ActionTypeTool, Value: "reject_request"},
    },
)
```

**Use Cases:**
- Present action choices
- Quick reply options
- Navigation menus
- Confirm/cancel dialogs

### 2. Timeline - Chronological Events

```go
type TimelinePart struct {
    Title       string
    Description string
    Events      []TimelineEvent
    Layout      TimelineLayout // default, alternate, compact, detailed
    Orientation string         // vertical, horizontal
}

type TimelineEvent struct {
    Title       string
    Description string
    Timestamp   time.Time
    Icon        string
    Color       string
    Status      TimelineStatus // pending, in_progress, completed, cancelled, error
    Link        string
    Tags        []string
    Actions     []Button
}
```

**Example:**
```go
timeline := NewTimelinePart(
    TimelineEvent{
        Title:       "Order Placed",
        Description: "Order #12345 was placed successfully",
        Timestamp:   time.Now(),
        Icon:        "shopping-cart",
        Status:      TimelineStatusCompleted,
    },
    TimelineEvent{
        Title:       "Payment Processed",
        Description: "Payment of $99.99 received",
        Timestamp:   time.Now().Add(5 * time.Minute),
        Icon:        "credit-card",
        Status:      TimelineStatusCompleted,
    },
)
```

**Use Cases:**
- Order tracking
- Project milestones
- Activity logs
- History views
- Deployment pipelines

### 3. Kanban - Task Boards

```go
type KanbanPart struct {
    Title       string
    Description string
    Columns     []KanbanColumn
    Draggable   bool
    ShowCounts  bool
}

type KanbanColumn struct {
    ID      string
    Title   string
    Color   string
    Cards   []KanbanCard
    Limit   int // WIP limit
}

type KanbanCard struct {
    Title       string
    Description string
    Labels      []KanbanLabel
    Assignees   []KanbanUser
    Priority    string  // low, medium, high, urgent
    Progress    float64 // 0-100
    DueDate     *time.Time
}
```

**Example:**
```go
kanban := NewKanbanPart(
    KanbanColumn{
        ID:    "todo",
        Title: "To Do",
        Color: "#gray",
        Cards: []KanbanCard{
            {
                Title:    "Implement login",
                Priority: "high",
                Labels:   []KanbanLabel{{Text: "backend", Color: "#blue"}},
            },
        },
    },
    KanbanColumn{
        ID:    "in_progress",
        Title: "In Progress",
        Color: "#yellow",
        Cards: []KanbanCard{
            {
                Title:    "Design dashboard",
                Priority: "medium",
                Progress: 60.0,
            },
        },
    },
)
```

**Use Cases:**
- Task management
- Workflow visualization
- Sprint boards
- Pipeline status
- Content planning

### 4. Metric - KPIs & Dashboards

```go
type MetricPart struct {
    Title       string
    Description string
    Metrics     []Metric
    Layout      MetricLayout // grid, list, compact, cards
    Columns     int
}

type Metric struct {
    Label          string
    Value          any
    FormattedValue string
    Unit           string
    Icon           string
    Color          string
    Trend          *MetricTrend
    Sparkline      []float64
    Target         *MetricTarget
    Status         MetricStatus // good, warning, bad, neutral
}

type MetricTrend struct {
    Direction  TrendDirection // up, down, stable
    Value      float64
    Percentage float64
    Period     string
}
```

**Example:**
```go
metrics := NewMetricPart(
    Metric{
        Label:          "Revenue",
        Value:          125000,
        FormattedValue: "$125,000",
        Icon:           "dollar-sign",
        Trend: &MetricTrend{
            Direction:  TrendUp,
            Percentage: 12.5,
            Period:     "vs last month",
        },
        Status: MetricStatusGood,
    },
    Metric{
        Label:          "Active Users",
        Value:          1842,
        FormattedValue: "1,842",
        Icon:           "users",
        Trend: &MetricTrend{
            Direction:  TrendUp,
            Percentage: 8.3,
        },
        Status: MetricStatusGood,
    },
)
```

**Use Cases:**
- Business dashboards
- Analytics summaries
- Performance monitoring
- KPI tracking
- Real-time metrics

### 5. Form - Interactive Input

```go
type FormPart struct {
    ID            string
    Title         string
    Description   string
    Fields        []FormField
    Sections      []FormSection
    Actions       []Button
    Layout        FormLayout // vertical, horizontal, inline, grid
    DefaultValues map[string]any
}

type FormField struct {
    ID           string
    Name         string
    Label        string
    Type         FormFieldType // text, email, password, number, select, checkbox, etc.
    Placeholder  string
    Required     bool
    Options      []FormFieldOption
    Validation   *FieldValidation
}
```

**Example:**
```go
form := NewFormPart("user_registration",
    FormField{
        Name:        "email",
        Label:       "Email Address",
        Type:        FieldTypeEmail,
        Placeholder: "you@example.com",
        Required:    true,
    },
    FormField{
        Name:     "role",
        Label:    "Role",
        Type:     FieldTypeSelect,
        Required: true,
        Options: []FormFieldOption{
            {Value: "admin", Label: "Administrator"},
            {Value: "user", Label: "Regular User"},
        },
    },
)
```

**Use Cases:**
- User registration
- Settings configuration
- Search filters
- Data entry
- Surveys

### 6. Chart - Data Visualization

```go
type ChartPart struct {
    ChartType ChartType // line, bar, pie, doughnut, area, scatter
    Title     string
    Data      ChartData
    Options   *ChartOptions
}

type ChartData struct {
    Labels   []string
    Datasets []ChartDataset
}

type ChartDataset struct {
    Label           string
    Data            []float64
    BackgroundColor string
    BorderColor     string
}
```

**Example:**
```go
chart := &ChartPart{
    ChartType: ChartLine,
    Title:     "Monthly Sales",
    Data: ChartData{
        Labels: []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"},
        Datasets: []ChartDataset{
            {
                Label: "2024",
                Data:  []float64{65, 59, 80, 81, 56, 55},
                BorderColor: "#4CAF50",
            },
            {
                Label: "2025",
                Data:  []float64{28, 48, 40, 19, 86, 27},
                BorderColor: "#2196F3",
            },
        },
    },
}
```

**Use Cases:**
- Trend visualization
- Comparative analysis
- Distribution charts
- Time series data
- Statistical reports

### 7. Table - Structured Data

```go
type TablePart struct {
    Title      string
    Caption    string
    Headers    []TableHeader
    Rows       [][]TableCell
    Sortable   bool
    Searchable bool
    Paginated  bool
    Style      *TableStyle
}

type TableCell struct {
    Value   any
    Display string
    Link    string
    Style   string
}
```

**Example:**
```go
table := &TablePart{
    Title: "User List",
    Headers: []TableHeader{
        {Label: "Name", Sortable: true},
        {Label: "Email", Sortable: true},
        {Label: "Status", Sortable: false},
    },
    Rows: [][]TableCell{
        {
            {Value: "Alice", Display: "Alice Johnson"},
            {Value: "alice@example.com", Display: "alice@example.com"},
            {Value: "active", Display: "Active", Style: "success"},
        },
    },
    Sortable:   true,
    Searchable: true,
    Style:      &TableStyle{Striped: true, Hover: true},
}
```

**Use Cases:**
- Data grids
- Search results
- Database records
- Comparison tables
- Reports

---

## Presentation Tools

Presentation tools are AI-callable functions that generate UI components. These tools are designed to be used by LLMs via function calling.

### Available Tools

| Tool | Description | Streaming | Output Type |
|------|-------------|-----------|-------------|
| `render_table` | Display tabular data | ✅ | `TablePart` |
| `render_chart` | Display charts | ❌ | `ChartPart` |
| `render_metrics` | Display KPIs | ✅ | `MetricPart` |
| `render_timeline` | Display events | ✅ | `TimelinePart` |
| `render_kanban` | Display board | ✅ | `KanbanPart` |
| `render_buttons` | Display actions | ❌ | `ButtonGroupPart` |
| `render_form` | Display form | ✅ | `FormPart` |
| `render_card` | Display card | ❌ | `CardPart` |
| `render_stats` | Display stats | ❌ | `StatsPart` |
| `render_gallery` | Display gallery | ❌ | `GalleryPart` |
| `render_alert` | Display alert | ❌ | `AlertPart` |
| `render_progress` | Display progress | ❌ | `ProgressPart` |

### Tool Usage Pattern

```
1. LLM decides which UI to generate
2. LLM calls presentation tool with parameters
3. Tool handler processes parameters
4. Render function generates ContentPart
5. ContentPart streamed to frontend
6. Frontend renders native UI component
```

### Getting Tool Schemas

```go
// Get all presentation tool schemas for LLM
schemas := sdk.GetPresentationToolSchemas()

// Add to agent tools
agent, _ := sdk.NewAgentBuilder().
    WithTools(schemas...).
    Build()
```

### Executing Presentation Tools

```go
// Execute a presentation tool
result, err := sdk.ExecutePresentationTool(
    ctx,
    "render_table",
    map[string]any{
        "title": "Sales Data",
        "headers": []string{"Product", "Sales", "Growth"},
        "rows": [][]any{
            {"Widget", 1000, "+15%"},
            {"Gadget", 2000, "+25%"},
        },
    },
    onStreamEvent,
    executionID,
)

// Result contains the rendered ContentPart
part := result.Part
```

---

## Usage Examples

### Example 1: Build Response Programmatically

```go
// Using ResponseBuilder for fluent API
response := sdk.NewResponseBuilder().
    WithMetadata("gpt-4", "openai", duration).
    AddText("Here's the analysis:").
    AddTable(
        []string{"Metric", "Value", "Change"},
        [][]any{
            {"Revenue", "$125K", "+12%"},
            {"Users", "1,842", "+8%"},
        },
    ).
    AddDivider().
    AddAlert("Processing complete!", sdk.AlertSuccess).
    Build()

// Convert to JSON for frontend
jsonData, _ := response.ToJSON()
```

### Example 2: Parse LLM Output into UI

```go
// LLM returns mixed content with code and text
llmOutput := `
Here's the implementation:

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `

This code prints a greeting.
`

// Parse into structured parts
parser := sdk.NewResponseParser()
parts := parser.Parse(llmOutput)

// Parts contains: CodePart and TextPart
for _, part := range parts {
    switch p := part.(type) {
    case *sdk.CodePart:
        fmt.Printf("Code: %s (Language: %s)\n", p.Code, p.Language)
    case *sdk.TextPart:
        fmt.Printf("Text: %s\n", p.Text)
    }
}
```

### Example 3: Agent with Presentation Tools

```go
// Create agent with presentation tools
agent, _ := sdk.NewAgentBuilder().
    WithName("data_analyst").
    WithModel("gpt-4").
    WithSystemPrompt(`You are a data analyst. Use presentation tools to visualize data:
- render_table for data grids
- render_chart for trends
- render_metrics for KPIs`).
    WithTools(sdk.GetPresentationToolSchemas()...).
    WithLLMManager(llmManager).
    Build()

// Agent automatically calls presentation tools
execution, _ := agent.Execute(ctx, "Show me user engagement metrics")

// Execution result contains rich UI parts
for _, part := range execution.Response.Parts {
    fmt.Printf("Part Type: %s\n", part.Type())
}
```

### Example 4: Streaming UI Generation

```go
// Stream complex UI progressively
streamer := sdk.NewUIPartStreamer(sdk.UIPartStreamerConfig{
    PartType: sdk.PartTypeTable,
    OnEvent: func(event llm.ClientStreamEvent) error {
        // Send updates to client in real-time
        return sendToClient(event)
    },
})

streamer.Start()

// Stream title
streamer.StreamSection("title", "Sales Report")

// Stream headers
streamer.StreamHeader([]sdk.TableHeader{
    {Label: "Product"},
    {Label: "Sales"},
})

// Stream rows in batches
for _, batch := range rowBatches {
    streamer.StreamRows(batch)
    time.Sleep(100 * time.Millisecond) // Simulate processing
}

streamer.End()
```

### Example 5: Multi-Part Dashboard

```go
// Build complex dashboard response
dashboard := sdk.NewResponseBuilder().
    AddText("# Q4 2025 Performance Dashboard").
    AddDivider().
    AddPart(sdk.NewMetricPart(
        sdk.Metric{
            Label: "Revenue",
            Value: 500000,
            Trend: &sdk.MetricTrend{Direction: sdk.TrendUp, Percentage: 25},
        },
        sdk.Metric{
            Label: "Customers",
            Value: 12450,
            Trend: &sdk.MetricTrend{Direction: sdk.TrendUp, Percentage: 15},
        },
    )).
    AddDivider().
    AddPart(&sdk.ChartPart{
        ChartType: sdk.ChartLine,
        Title:     "Monthly Trends",
        Data:      chartData,
    }).
    AddDivider().
    AddTable(detailedDataHeaders, detailedDataRows).
    Build()
```

---

## Integration Guide

### Frontend Integration

#### React Example

```typescript
import React from 'react';

interface ContentPart {
  type: string;
  [key: string]: any;
}

function ContentPartRenderer({ part }: { part: ContentPart }) {
  switch (part.type) {
    case 'table':
      return <TableComponent {...part} />;
    case 'chart':
      return <ChartComponent {...part} />;
    case 'button_group':
      return <ButtonGroup buttons={part.buttons} />;
    case 'timeline':
      return <Timeline events={part.events} />;
    case 'kanban':
      return <KanbanBoard columns={part.columns} />;
    case 'metric':
      return <MetricsDashboard metrics={part.metrics} />;
    case 'form':
      return <DynamicForm fields={part.fields} />;
    case 'text':
      return <Markdown content={part.text} />;
    default:
      return <div>Unsupported part type: {part.type}</div>;
  }
}

function StructuredResponse({ response }: { response: StructuredResponse }) {
  return (
    <div>
      {response.parts.map((part, idx) => (
        <ContentPartRenderer key={idx} part={part} />
      ))}
    </div>
  );
}
```

#### Vue Example

```vue
<template>
  <div>
    <component
      v-for="(part, idx) in response.parts"
      :key="idx"
      :is="getComponent(part.type)"
      v-bind="part"
    />
  </div>
</template>

<script>
export default {
  props: ['response'],
  methods: {
    getComponent(type) {
      const components = {
        table: 'TableComponent',
        chart: 'ChartComponent',
        button_group: 'ButtonGroup',
        timeline: 'Timeline',
        kanban: 'KanbanBoard',
        metric: 'MetricsDashboard',
        form: 'DynamicForm',
        text: 'TextRenderer',
      };
      return components[type] || 'UnknownComponent';
    },
  },
};
</script>
```

### Backend Integration

```go
// Create an endpoint that returns structured responses
func handleChat(w http.ResponseWriter, r *http.Request) {
    var req ChatRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Execute agent with presentation tools
    execution, err := agent.Execute(r.Context(), req.Message)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Build structured response
    response := execution.Response // Already a StructuredResponse
    
    // Add metadata
    response.Metadata.Duration = execution.Duration
    response.Metadata.InputTokens = execution.TokensUsed.Input
    response.Metadata.OutputTokens = execution.TokensUsed.Output
    
    // Return as JSON
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

---

## Best Practices

### 1. Choose the Right Component

```go
// ✅ Good: Use appropriate component for data type
// Table for structured data
render_table(headers, rows)

// Chart for trends
render_chart("line", labels, datasets)

// Metrics for KPIs
render_metrics(metrics)

// ❌ Bad: Using text for everything
return "Revenue: $125K (+12%), Users: 1842 (+8%)"
```

### 2. Progressive Disclosure

```go
// ✅ Good: Use collapsible/tabs for large content
response := NewResponseBuilder().
    AddText("Summary: ...").
    AddPart(NewAccordionPart(
        AccordionItem{
            Title: "Detailed Analysis",
            Content: []ContentPart{/* detailed parts */},
        },
    )).
    Build()

// ❌ Bad: Dumping everything at once
response := NewResponseBuilder().
    AddText("... 5000 lines of detailed analysis ...").
    Build()
```

### 3. Actionable UIs

```go
// ✅ Good: Provide next actions
response.AddPart(NewButtonGroupPart(
    Button{Label: "Export to CSV", Action: ButtonAction{Type: ActionTypeDownload}},
    Button{Label: "Refresh Data", Action: ButtonAction{Type: ActionTypeTool, Value: "refresh"}},
))

// ❌ Bad: Static, non-interactive
response.AddText("You can export this to CSV")
```

### 4. Streaming for Large Data

```go
// ✅ Good: Stream large tables progressively
streamer.Start()
for _, batch := range rowBatches {
    streamer.StreamRows(batch)
}
streamer.End()

// ❌ Bad: Sending all at once
response.AddTable(headers, allMillionRows) // Don't do this!
```

### 5. Consistent Styling

```go
// ✅ Good: Use consistent colors/icons
Metric{
    Status: MetricStatusGood,
    Color:  "#4CAF50",
}

// ❌ Bad: Random colors
Metric{
    Status: MetricStatusGood,
    Color:  "#FF69B4", // Pink for "good"?
}
```

### 6. Accessibility

```go
// ✅ Good: Include labels, tooltips, alt text
Button{
    Label:   "Delete",
    Tooltip: "Permanently delete this item",
    Icon:    "trash",
}

ImagePart{
    URL: "chart.png",
    Alt: "Bar chart showing monthly sales",
}

// ❌ Bad: Icon-only with no labels
Button{
    Icon: "trash", // What does this do?
}
```

### 7. Error Handling

```go
// ✅ Good: Show errors as alerts
if err != nil {
    response.AddAlert(
        fmt.Sprintf("Error: %s", err.Error()),
        AlertError,
    )
}

// ❌ Bad: Silent failures or raw errors
return "", err // User sees nothing
```

### 8. Mobile-Friendly

```go
// ✅ Good: Use responsive layouts
MetricPart{
    Layout:  MetricLayoutGrid,
    Columns: 3, // Will adapt on mobile
}

// ❌ Bad: Fixed widths
TablePart{
    Headers: []TableHeader{
        {Width: "500px"}, // Too wide for mobile
    },
}
```

---

## Advanced Topics

### Custom ContentParts

You can create custom ContentPart types:

```go
type CustomPart struct {
    PartType ContentPartType `json:"type"`
    // Custom fields
    Data map[string]any `json:"data"`
}

func (c *CustomPart) Type() ContentPartType {
    return ContentPartType("custom")
}

func (c *CustomPart) ToJSON() ([]byte, error) {
    c.PartType = ContentPartType("custom")
    return json.Marshal(c)
}
```

### UI Block Syntax

Enable parsing of UI blocks in LLM output:

```go
parser := sdk.NewResponseParser().WithUIBlocks(true)

// LLM can output:
// ```ui:table
// {
//   "headers": ["Name", "Value"],
//   "rows": [["Item", "100"]]
// }
// ```

parts := parser.Parse(llmOutput) // Automatically parsed as TablePart
```

### Metadata and Context

Add context to parts for frontend:

```go
part := &TablePart{
    Title: "Users",
    Metadata: map[string]any{
        "data_source": "database",
        "last_updated": time.Now(),
        "total_count": 1000,
        "filtered_count": 50,
    },
}
```

---

## Troubleshooting

### Problem: Frontend Can't Render Part

**Solution**: Check that part type is supported in frontend:

```typescript
// Add fallback for unsupported types
function ContentPartRenderer({ part }) {
  const component = COMPONENT_MAP[part.type];
  if (!component) {
    console.warn(`Unsupported part type: ${part.type}`);
    return <UnknownPartRenderer part={part} />;
  }
  return React.createElement(component, part);
}
```

### Problem: Large Data Causes Performance Issues

**Solution**: Use pagination, virtualization, or streaming:

```go
// Backend: Paginate data
TablePart{
    Paginated: true,
    PageSize:  50,
    Rows:      firstPage,
}

// Frontend: Implement virtual scrolling
<VirtualizedTable data={table.rows} />
```

### Problem: Buttons Not Working

**Solution**: Ensure action handlers are implemented:

```typescript
function ButtonComponent({ button }) {
  const handleClick = () => {
    switch (button.action.type) {
      case 'tool':
        executeTool(button.action.value, button.action.payload);
        break;
      case 'link':
        window.location.href = button.action.value;
        break;
      case 'copy':
        navigator.clipboard.writeText(button.action.value);
        break;
    }
  };
  return <button onClick={handleClick}>{button.label}</button>;
}
```

---

## Performance Tips

1. **Stream Large Data**: Use `StreamRows()` for tables with > 100 rows
2. **Lazy Load**: Use accordions/tabs for content not immediately needed
3. **Paginate**: Set `Paginated: true` for large tables
4. **Limit Charts**: Keep chart datasets under 1000 points
5. **Optimize Images**: Use thumbnails in galleries, full res on click
6. **Batch Updates**: Group multiple content parts in single response

---

## Next Steps

- [Agent Patterns Overview](./AGENT_PATTERNS.md) - Use UI generation in agents
- [ReAct Agents Guide](./REACT_AGENTS.md) - ReAct with UI output
- [Plan-Execute Agents Guide](./PLAN_EXECUTE_AGENTS.md) - Plans as kanban/timeline
- [Streaming Guide](./STREAMING.md) - Progressive UI rendering

---

## Reference

### All ContentPart Types

<details>
<summary>Complete list of all 25+ ContentPart types</summary>

- `PartTypeText` - Plain or formatted text
- `PartTypeMarkdown` - Markdown content
- `PartTypeCode` - Syntax-highlighted code
- `PartTypeTable` - Tabular data with sorting/filtering
- `PartTypeCard` - Card with title, content, actions
- `PartTypeList` - Ordered/unordered/checkbox lists
- `PartTypeImage` - Images with captions
- `PartTypeThinking` - AI reasoning/thought process
- `PartTypeQuote` - Quotations and callouts
- `PartTypeDivider` - Visual separator
- `PartTypeAlert` - Notifications (info/success/warning/error)
- `PartTypeProgress` - Progress bars and indicators
- `PartTypeChart` - Line, bar, pie, doughnut, area, scatter
- `PartTypeJSON` - Formatted JSON display
- `PartTypeCollapsible` - Expandable content
- `PartTypeButtonGroup` - Interactive button groups
- `PartTypeTimeline` - Chronological events
- `PartTypeKanban` - Board with draggable cards
- `PartTypeMetric` - KPIs with trends and sparklines
- `PartTypeForm` - Interactive forms with validation
- `PartTypeTabs` - Tabbed content sections
- `PartTypeAccordion` - Collapsible accordion
- `PartTypeInlineCitation` - Inline source references
- `PartTypeStats` - Compact statistics display
- `PartTypeCarousel` - Sliding content carousel
- `PartTypeGallery` - Image/media gallery

</details>

### All Presentation Tools

<details>
<summary>Complete list of all 12 presentation tools</summary>

1. **render_table** - Tabular data display
2. **render_chart** - Data visualization (6 chart types)
3. **render_metrics** - KPI dashboard with trends
4. **render_timeline** - Chronological event display
5. **render_kanban** - Task board with columns/cards
6. **render_buttons** - Interactive action buttons
7. **render_form** - Dynamic input forms
8. **render_card** - Content cards
9. **render_stats** - Compact statistics
10. **render_gallery** - Image/media gallery
11. **render_alert** - Notifications and alerts
12. **render_progress** - Progress indicators

</details>

---

**Built with ❤️ for creating beautiful AI-generated UIs**


