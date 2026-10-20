package irrl

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
	"github.com/anthropics/anthropic-sdk-go/toolrunner"
)

const (
	chatModel        = "claude-opus-5-5"
	chatMaxRows      = 20
	chatSystemPrompt = `You are the voice assistant for the owner of a scaffolding and equipment rental business.
The owner asks about performance, income, customers, payments, equipment and returns.

How the data works:
- Rental income is earned per day: each rented unit earns its daily rent for every day it is out, up to its planned end date. A period's income comes from business_snapshot or income_by_equipment for that period.
- Paid on an order = advance + completed invoices. Owed = order amount minus paid.
- Invoices have no payment date and no due date, so "overdue payment" means money still owed on an order older than N days (use receivables with min_age_days, 30 if not specified).
- The only recorded cost is repair cost. "Profit" can only mean rental income minus repair cost; say so when you use it.
- Not recorded at all: general business expenses (salaries, fuel, rent), cash or bank balance, equipment purchase price, maintenance schedules. For questions that need these, say plainly that this information is not recorded, and offer what the data can tell instead.

How to answer:
- Use tools for every number; never guess. Compare periods by calling a tool once per period.
- For forecasts or advice (next month's revenue, pricing, what to buy or replace), base it on the trends and utilization the tools return and say it is an estimate.
- Questions may be in Malayalam, Hindi, English or another Indian language. Write the answer in the language the request names, in its own script, keeping customer names, order numbers and product codes as they appear in the data.
- Amounts are Indian rupees: write ₹ with Indian digit grouping (₹1,25,000).
- The answer is converted to speech, so it is read aloud: plain sentences, no markdown, lists, tables or emoji. Give the direct answer first. Keep simple questions to one or two sentences; broad questions such as "how is my business doing" or "what needs my attention" may use up to five short sentences covering the most important points.`
)

// ist is India Standard Time; a fixed zone avoids depending on tzdata in the container.
var ist = time.FixedZone("IST", 5*60*60+30*60)

// ChatBot answers natural-language questions about the business using fixed database queries as tools.
type ChatBot struct {
	client  anthropic.Client
	service Service
	tools   []anthropic.BetaTool
}

func NewChatBot(apiKey string, service Service) (*ChatBot, error) {
	bot := &ChatBot{
		client:  anthropic.NewClient(option.WithAPIKey(apiKey)),
		service: service,
	}
	tools, err := bot.buildTools()
	if err != nil {
		return nil, err
	}
	bot.tools = tools
	return bot, nil
}

// Answer runs the tool loop for one question and returns Claude's final reply,
// written in replyLanguage (e.g. "Malayalam").
func (b *ChatBot) Answer(ctx context.Context, question, replyLanguage string) (string, error) {
	today := time.Now().In(ist).Format("Monday, 2006-01-02")
	prompt := fmt.Sprintf("Today is %s.\nWrite the answer in %s.\n\nQuestion: %s", today, replyLanguage, question)

	runner := b.client.Beta.Messages.NewToolRunner(b.tools, anthropic.BetaToolRunnerParams{
		BetaMessageNewParams: anthropic.BetaMessageNewParams{
			Model:        chatModel,
			MaxTokens:    16000,
			System:       []anthropic.BetaTextBlockParam{{Text: chatSystemPrompt}},
			OutputConfig: anthropic.BetaOutputConfigParam{Effort: anthropic.BetaOutputConfigEffortMedium},
			Fallbacks:    anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()},
			Betas:        []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
			Messages: []anthropic.BetaMessageParam{
				anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(prompt)),
			},
		},
		MaxIterations: 8,
	})

	message, err := runner.RunToCompletion(ctx)
	if err != nil {
		return "", fmt.Errorf("chat failed: %w", err)
	}
	if message.StopReason == anthropic.BetaStopReasonRefusal {
		return "", fmt.Errorf("the question was declined")
	}

	var answer strings.Builder
	for _, block := range message.Content {
		if text, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			answer.WriteString(text.Text)
		}
	}
	return strings.TrimSpace(answer.String()), nil
}

type noInput struct{}

type periodInput struct {
	From string `json:"from" jsonschema:"required,description=First day of the period, YYYY-MM-DD"`
	To   string `json:"to" jsonschema:"required,description=Last day of the period (inclusive), YYYY-MM-DD"`
}

type equipmentIncomeInput struct {
	From  string `json:"from" jsonschema:"required,description=First day of the period, YYYY-MM-DD"`
	To    string `json:"to" jsonschema:"required,description=Last day of the period (inclusive), YYYY-MM-DD"`
	Order string `json:"order" jsonschema:"enum=highest,enum=lowest,description=highest income first (default) or lowest income first"`
}

type limitInput struct {
	Limit int `json:"limit" jsonschema:"description=How many rows to return (1-20). Use 1 for 'the highest'."`
}

type daysInput struct {
	Days int `json:"days" jsonschema:"required,description=Number of days"`
}

type customerInput struct {
	Name  string `json:"name" jsonschema:"required,description=Full or partial customer name"`
	Limit int    `json:"limit" jsonschema:"description=How many recent orders to return (1-20)"`
}

type customerActivityInput struct {
	Sort string `json:"sort" jsonschema:"required,enum=revenue,enum=inactive,description=revenue: highest lifetime revenue first. inactive: longest time since last order first"`
}

type revenueTrendInput struct {
	Period string `json:"period" jsonschema:"required,enum=daily,enum=weekly,enum=monthly,description=Bucket size"`
	Limit  int    `json:"limit" jsonschema:"description=How many recent periods to return (1-20)"`
}

func (b *ChatBot) buildTools() ([]anthropic.BetaTool, error) {
	s := b.service
	builders := []func() (anthropic.BetaTool, error){
		newChatTool("business_snapshot",
			"Headline numbers for a period: rental income earned, orders placed and their value, advances received, cancelled orders, repair events and repair cost. "+
				"Also the current state: units on rent and available, units overdue for return, total owed by customers and how many customers owe. "+
				"Use for today, this month, last month, and period comparisons (call once per period).",
			func(in periodInput) (any, error) {
				from, to, err := parsePeriod(in.From, in.To)
				if err != nil {
					return nil, err
				}
				return s.GetBusinessSnapshot(from, to)
			}),
		newChatTool("income_by_equipment",
			"Rental income, rented unit-days and utilization % per product for a period, with units owned. Use for most/least earning equipment, demand and utilization.",
			func(in equipmentIncomeInput) (any, error) {
				from, to, err := parsePeriod(in.From, in.To)
				if err != nil {
					return nil, err
				}
				return s.GetIncomeByEquipment(from, to, in.Order == "lowest")
			}),
		newChatTool("fleet_status",
			"Current unit counts per product: total, available, rented, reserved, at site, damaged, repairing, blocked, worn out, and utilization now. The first row is the total for all products.",
			func(in noInput) (any, error) { return s.GetFleetStatus() }),
		newChatTool("idle_equipment",
			"Per product, available units that have not been rented in the last N days (including never rented), with the oldest last-rented date.",
			func(in daysInput) (any, error) { return s.GetIdleEquipment(clampDays(in.Days)) }),
		newChatTool("receivables",
			"Money owed per customer on orders older than min_age_days (0 for everything), highest first, with pending invoice amounts and the age of the oldest unpaid order. The first row is the total for all customers.",
			func(in struct {
				MinAgeDays int `json:"min_age_days" jsonschema:"required,description=Only count orders placed more than this many days ago. 0 for all."`
			}) (any, error) {
				return s.GetReceivables(clampDays(in.MinAgeDays))
			}),
		newChatTool("customer_payment_history",
			"Recent orders of a customer (matched by partial name) with order amount, advance, invoices paid and pending, amount owed and order age.",
			func(in customerInput) (any, error) {
				return s.GetCustomerPaymentHistory(in.Name, clampLimit(in.Limit))
			}),
		newChatTool("largest_orders",
			"Individual orders ranked by order amount, largest first, with customer, amount, paid, owed, status and date.",
			func(in limitInput) (any, error) { return s.GetLargestOrders(clampLimit(in.Limit)) }),
		newChatTool("rental_returns",
			"Rented units past their planned return date, plus units due back within the next N days (0 for overdue only). "+
				"Includes days overdue (negative means due in future), daily rent and the rent not yet billed for overdue days.",
			func(in daysInput) (any, error) { return s.GetRentalReturns(clampDays(in.Days)) }),
		newChatTool("customer_activity",
			"Per customer: lifetime orders, revenue, amount owed, last order date and days since last order. Use for most valuable, lapsed and follow-up customers.",
			func(in customerActivityInput) (any, error) { return s.GetCustomerActivity(in.Sort) }),
		newChatTool("repair_vs_income",
			"Per product for a period: repair events, units repaired, repair cost, rental income, repair cost as % of income, and units damaged or in repair now. Only products with repairs or damage are listed.",
			func(in periodInput) (any, error) {
				from, to, err := parsePeriod(in.From, in.To)
				if err != nil {
					return nil, err
				}
				return s.GetRepairVsIncome(from, to)
			}),
		newChatTool("revenue_trend",
			"Value of orders placed per day, week or month for recent periods. Useful for trends and rough forecasts.",
			func(in revenueTrendInput) (any, error) { return s.GetRevenueTrend(in.Period, clampLimit(in.Limit)) }),
		newChatTool("order_status_summary",
			"Number of orders in each status.",
			func(in noInput) (any, error) { return s.GetOrderStatusSummary() }),
		newChatTool("customer_damage_stats",
			"Damage counts per customer.",
			func(in noInput) (any, error) { return s.GetCustomerDamageAnalytics() }),
	}

	tools := make([]anthropic.BetaTool, 0, len(builders))
	for _, build := range builders {
		tool, err := build()
		if err != nil {
			return nil, fmt.Errorf("failed to build chat tool: %w", err)
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// newChatTool wraps a query function as a tool whose result is JSON text, logging each call.
func newChatTool[T any](name, description string, run func(T) (any, error)) func() (anthropic.BetaTool, error) {
	return func() (anthropic.BetaTool, error) {
		return toolrunner.NewBetaToolFromJSONSchema(name, description,
			func(ctx context.Context, in T) (anthropic.BetaToolResultBlockParamContentUnion, error) {
				return toolResult(name, in)(run(in))
			})
	}
}

// parsePeriod validates an inclusive YYYY-MM-DD range and returns it as [from, to) with
// `to` capped at tomorrow, so periods that run into the future only count days so far.
func parsePeriod(from, to string) (string, string, error) {
	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return "", "", fmt.Errorf("from must be YYYY-MM-DD, got %q", from)
	}
	end, err := time.Parse("2006-01-02", to)
	if err != nil {
		return "", "", fmt.Errorf("to must be YYYY-MM-DD, got %q", to)
	}
	end = end.AddDate(0, 0, 1)

	now := time.Now().In(ist)
	tomorrow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
	if end.After(tomorrow) {
		end = tomorrow
	}
	if !end.After(start) {
		return "", "", fmt.Errorf("the period %s to %s has no days up to today", from, to)
	}
	return start.Format("2006-01-02"), end.Format("2006-01-02"), nil
}

func clampLimit(limit int) int {
	if limit <= 0 || limit > chatMaxRows {
		return chatMaxRows
	}
	return limit
}

func clampDays(days int) int {
	if days < 0 {
		return 0
	}
	if days > 3650 {
		return 3650
	}
	return days
}

// toolResult logs the call and turns a query result into JSON text, keeping at most chatMaxRows rows.
func toolResult(name string, input any) func(any, error) (anthropic.BetaToolResultBlockParamContentUnion, error) {
	return func(data any, err error) (anthropic.BetaToolResultBlockParamContentUnion, error) {
		log.Printf("chat tool %s input=%+v err=%v", name, input, err)
		if err != nil {
			return anthropic.BetaToolResultBlockParamContentUnion{}, err
		}

		var payload any = data
		var rows []json.RawMessage
		if raw, mErr := json.Marshal(data); mErr == nil && json.Unmarshal(raw, &rows) == nil && len(rows) > chatMaxRows {
			payload = map[string]any{
				"rows":       rows[:chatMaxRows],
				"total_rows": len(rows),
				"note":       fmt.Sprintf("only the first %d rows are shown", chatMaxRows),
			}
		}

		out, err := json.Marshal(payload)
		if err != nil {
			return anthropic.BetaToolResultBlockParamContentUnion{}, err
		}
		return anthropic.BetaToolResultBlockParamContentUnion{
			OfText: &anthropic.BetaTextBlockParam{Text: string(out)},
		}, nil
	}
}
