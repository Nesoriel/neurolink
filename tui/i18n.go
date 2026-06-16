package tui

import (
	"fmt"
	"strings"

	"github.com/Nesoriel/neurolink/statusapi"
)

type Language string

const (
	LanguageEnglish           Language = "en"
	LanguageSimplifiedChinese Language = "zh-Hans"
)

func ParseLanguage(value string) (Language, error) {
	value = strings.TrimSpace(value)
	switch value {
	case "", string(LanguageEnglish):
		return LanguageEnglish, nil
	case string(LanguageSimplifiedChinese):
		return LanguageSimplifiedChinese, nil
	default:
		return "", fmt.Errorf("unsupported language %q (supported: en, zh-Hans)", value)
	}
}

type lexicon struct {
	language Language

	stopped          string
	loading          string
	headerSubtitle   string
	idleMode         string
	battleMode       string
	lastUpdatePrefix string

	statusSummaryTitle string
	impactedServices   string
	downDegraded       string
	unknownChecks      string
	apiMode            string
	demoDataNotLive    string

	communityPulseTitle string
	refreshIssuePrefix  string
	noReportsLine1      string
	noReportsLine2      string
	attributionPrefix   string
	reportReceived      string

	noRegionalDetail string
	updatedPrefix    string

	footerSetup string
	footerLive  string

	running  string
	degraded string
	down     string
	unknown  string

	apiKeyRejected  string
	refreshCanceled string
}

func lexiconFor(language Language) lexicon {
	if language == LanguageSimplifiedChinese {
		return lexicon{
			language: LanguageSimplifiedChinese,

			stopped:          "neurolink 已停止\n",
			loading:          "NEUROLINK // 正在获取服务状态...",
			headerSubtitle:   "Crypto 监控无人机 / Apex 服务健康",
			idleMode:         "待机",
			battleMode:       "战斗",
			lastUpdatePrefix: "更新于 ",

			statusSummaryTitle: "状态总览",
			impactedServices:   "受影响服务",
			downDegraded:       "中断 / 降级",
			unknownChecks:      "未知检查",
			apiMode:            "API 模式",
			demoDataNotLive:    "演示数据不是实时 Apex 服务状态。",

			communityPulseTitle: "用户反馈",
			refreshIssuePrefix:  "刷新问题：",
			noReportsLine1:      "/servers 可能不包含用户反馈 feed。",
			noReportsLine2:      "不查询玩家资料或玩家状态。",
			attributionPrefix:   "来源：",
			reportReceived:      "已收到反馈",

			noRegionalDetail: "没有区域明细",
			updatedPrefix:    "更新于 ",

			footerSetup: "q 退出  ctrl+c 退出  config set api-key 或 --api-key 启用实时状态  --demo 使用示例数据",
			footerLive:  "q 退出  ctrl+c 退出  --poll-interval 控制刷新频率",

			running:  "正常",
			degraded: "降级",
			down:     "中断",
			unknown:  "未知",

			apiKeyRejected:  "状态 API 拒绝了 API key",
			refreshCanceled: "刷新已取消",
		}
	}

	return lexicon{
		language: LanguageEnglish,

		stopped:          "neurolink stopped\n",
		loading:          "NEUROLINK // acquiring service feed...",
		headerSubtitle:   "Crypto surveillance drone / Apex service health",
		idleMode:         "IDLE",
		battleMode:       "BATTLE",
		lastUpdatePrefix: "Last update ",

		statusSummaryTitle: "STATUS SUMMARY",
		impactedServices:   "Impacted services",
		downDegraded:       "Down / degraded",
		unknownChecks:      "Unknown checks",
		apiMode:            "API mode",
		demoDataNotLive:    "Demo data is not live Apex service status.",

		communityPulseTitle: "COMMUNITY PULSE",
		refreshIssuePrefix:  "Refresh issue: ",
		noReportsLine1:      "/servers may not include a report feed.",
		noReportsLine2:      "No player profile or player-status lookup is performed.",
		attributionPrefix:   "Source: ",
		reportReceived:      "Report received",

		noRegionalDetail: "No regional detail",
		updatedPrefix:    "Updated ",

		footerSetup: "q quit  ctrl+c quit  config set api-key or --api-key for live status  --demo for sample data",
		footerLive:  "q quit  ctrl+c quit  --poll-interval controls refresh cadence",

		running:  "RUNNING",
		degraded: "DEGRADED",
		down:     "DOWN",
		unknown:  "UNKNOWN",

		apiKeyRejected:  "API key rejected by status API",
		refreshCanceled: "refresh canceled",
	}
}

func (c lexicon) statusLabel(status statusapi.Status) string {
	switch status {
	case statusapi.StatusHealthy:
		return "● " + c.running
	case statusapi.StatusDegraded:
		return "▲ " + c.degraded
	case statusapi.StatusDown:
		return "✕ " + c.down
	default:
		return "? " + c.unknown
	}
}

func (c lexicon) modeLabel(battle bool) string {
	if battle {
		return "● " + c.battleMode
	}
	return "● " + c.idleMode
}

func (c lexicon) notice(value string) string {
	value = strings.TrimSpace(value)
	if c.language != LanguageSimplifiedChinese {
		return value
	}

	switch value {
	case "Demo mode: deterministic sample data, not live Apex service status":
		return "演示模式：确定性的示例数据，不是实时 Apex 服务状态"
	case "Demo data is not live Apex service status.":
		return c.demoDataNotLive
	case "Unable to refresh service data":
		return "无法刷新服务数据"
	default:
		return value
	}
}

func (c lexicon) attributionLine(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if c.language == LanguageSimplifiedChinese {
		value = strings.TrimPrefix(value, "Data from ")
	}
	return c.attributionPrefix + value
}

func (c lexicon) summarizeError(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "HTTP 401") || strings.Contains(value, "HTTP 403") {
		return c.apiKeyRejected
	}
	if strings.Contains(value, "context canceled") {
		return c.refreshCanceled
	}
	return fit(value, 72)
}
