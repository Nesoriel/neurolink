package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Nesoriel/neurolink/playerapi"
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
	refreshAge         string
	historyTrend       string
	refreshQueued      string
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

	playerLookupTitle    string
	playerLookupSubtitle string
	playerInputLabel     string
	playerPlaceholder    string
	playerPlatformLabel  string
	pcOriginNote         string
	playerLoading        string
	playerResultTitle    string
	playerErrorTitle     string
	playerIdleLine1      string
	playerIdleLine2      string
	playerSource         string
	playerUID            string
	playerPlatform       string
	playerLevel          string
	playerRank           string
	playerLegend         string
	playerTrackers       string

	configTitle    string
	configSource   string
	configLanguage string
	configAPIKey   string
	configEnv      string
	configDemo     string

	helpTitle     string
	helpDashboard string
	helpPlayer    string
	helpConfig    string
	helpHelp      string
	helpSearch    string
	helpPlatform  string
	helpRefresh   string
	helpQuit      string
	helpPrivacy   string

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
			refreshAge:         "刷新距今",
			historyTrend:       "趋势",
			refreshQueued:      "已请求刷新，正在等待状态 feed 返回。",
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

			footerSetup: "tab 切换  1 状态  2 玩家  3 配置  ? 帮助  / 输入玩家  enter 查询  r 刷新  q 退出",
			footerLive:  "tab 切换  1 状态  2 玩家  3 配置  ? 帮助  / 输入玩家  enter 查询  p 平台  r 刷新  q 退出",

			playerLookupTitle:    "玩家查询",
			playerLookupSubtitle: "按玩家名和平台显式查询 /bridge；不会后台跟踪玩家。",
			playerInputLabel:     "玩家名",
			playerPlaceholder:    "输入 Origin / PSN / Xbox 名称",
			playerPlatformLabel:  "平台",
			pcOriginNote:         "PC 查询通常使用 Origin 账号名，即使该账号通过 Steam 游玩。",
			playerLoading:        "正在查询玩家资料...",
			playerResultTitle:    "查询结果",
			playerErrorTitle:     "查询错误",
			playerIdleLine1:      "输入玩家名并选择平台，然后按 Enter。",
			playerIdleLine2:      "支持 PC、PS4、X1；查询只会在你按下 Enter 时运行。",
			playerSource:         "来源",
			playerUID:            "UID",
			playerPlatform:       "平台",
			playerLevel:          "等级",
			playerRank:           "排位",
			playerLegend:         "当前传奇",
			playerTrackers:       "追踪器",

			configTitle:    "配置",
			configSource:   "当前状态源",
			configLanguage: "语言",
			configAPIKey:   "API key：使用 --api-key、NEUROLINK_APEX_API_KEY，或 neurolink config set api-key 保存。",
			configEnv:      "语言：--lang 或 NEUROLINK_LANG，支持 en 和 zh-Hans。",
			configDemo:     "无 API key 时使用明确标记的 demo 数据；玩家查询同样返回 demo 结果。",

			helpTitle:     "帮助",
			helpDashboard: "1 / tab：状态仪表盘",
			helpPlayer:    "2 / /：玩家查询视图并聚焦输入框",
			helpConfig:    "3：配置视图",
			helpHelp:      "?：帮助视图",
			helpSearch:    "enter：在玩家视图中按当前输入和平台查询",
			helpPlatform:  "p：玩家输入框未聚焦时切换 PC / PS4 / X1",
			helpRefresh:   "r：立即请求刷新服务状态",
			helpQuit:      "q / ctrl+c：退出",
			helpPrivacy:   "玩家数据只在显式查询时请求；neurolink 不做连续玩家跟踪或隐藏遥测。",

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
		refreshAge:         "Refresh age",
		historyTrend:       "Trend",
		refreshQueued:      "Refresh requested; waiting for the status feed.",
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

		footerSetup: "tab views  1 dashboard  2 player  3 config  ? help  / player input  enter lookup  r refresh  q quit",
		footerLive:  "tab views  1 dashboard  2 player  3 config  ? help  / player input  enter lookup  p platform  r refresh  q quit",

		playerLookupTitle:    "PLAYER LOOKUP",
		playerLookupSubtitle: "Explicit /bridge lookup by player name and platform; no background player tracking.",
		playerInputLabel:     "Player name",
		playerPlaceholder:    "Origin / PSN / Xbox name",
		playerPlatformLabel:  "Platform",
		pcOriginNote:         "PC lookup generally uses the Origin account name, even for Steam-linked accounts.",
		playerLoading:        "Looking up player...",
		playerResultTitle:    "PLAYER RESULT",
		playerErrorTitle:     "LOOKUP ERROR",
		playerIdleLine1:      "Type a player name, choose a platform, then press Enter.",
		playerIdleLine2:      "Supports PC, PS4, and X1; lookup only runs after your visible action.",
		playerSource:         "Source",
		playerUID:            "UID",
		playerPlatform:       "Platform",
		playerLevel:          "Level",
		playerRank:           "Rank",
		playerLegend:         "Selected legend",
		playerTrackers:       "Trackers",

		configTitle:    "CONFIG",
		configSource:   "Current status source",
		configLanguage: "Language",
		configAPIKey:   "API key: use --api-key, NEUROLINK_APEX_API_KEY, or neurolink config set api-key.",
		configEnv:      "Language: --lang or NEUROLINK_LANG; supported values are en and zh-Hans.",
		configDemo:     "Without an API key, clearly labeled demo data is used; player lookup also returns demo results.",

		helpTitle:     "HELP",
		helpDashboard: "1 / tab: dashboard",
		helpPlayer:    "2 / /: player lookup view and focused input",
		helpConfig:    "3: config view",
		helpHelp:      "?: help view",
		helpSearch:    "enter: search from the player view",
		helpPlatform:  "p: cycle PC / PS4 / X1 when the player input is not focused",
		helpRefresh:   "r: request an immediate service refresh",
		helpQuit:      "q / ctrl+c: quit",
		helpPrivacy:   "Player data is requested only after explicit lookup; neurolink does not continuously track players or hide telemetry.",

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

func (c lexicon) viewLabel(view appView) string {
	switch view {
	case viewPlayer:
		if c.language == LanguageSimplifiedChinese {
			return "玩家"
		}
		return "PLAYER"
	case viewConfig:
		if c.language == LanguageSimplifiedChinese {
			return "配置"
		}
		return "CONFIG"
	case viewHelp:
		if c.language == LanguageSimplifiedChinese {
			return "帮助"
		}
		return "HELP"
	default:
		if c.language == LanguageSimplifiedChinese {
			return "状态"
		}
		return "DASHBOARD"
	}
}

func (c lexicon) viewKeyLabel(view appView) string {
	switch view {
	case viewPlayer:
		return "2 " + c.viewLabel(view)
	case viewConfig:
		return "3 " + c.viewLabel(view)
	case viewHelp:
		return "? " + c.viewLabel(view)
	default:
		return "1 " + c.viewLabel(view)
	}
}

func (c lexicon) footerHint(view appView, source statusapi.SourceMode) string {
	if source == statusapi.SourceLive {
		return c.footerLive
	}
	return c.footerSetup
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
		if strings.HasPrefix(value, "Demo player lookup") {
			return "演示玩家查询结果，不是实时 Apex 玩家数据。"
		}
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

func (c lexicon) playerError(err error) string {
	switch {
	case errors.Is(err, playerapi.ErrAPIKeyRequired):
		if c.language == LanguageSimplifiedChinese {
			return "未配置 API key。请使用 --api-key、NEUROLINK_APEX_API_KEY 或 config set api-key。"
		}
		return "API key is not configured. Use --api-key, NEUROLINK_APEX_API_KEY, or config set api-key."
	case errors.Is(err, playerapi.ErrInvalidAPIKey):
		if c.language == LanguageSimplifiedChinese {
			return "玩家查询 API 拒绝了 API key。"
		}
		return "Player lookup API rejected the API key."
	case errors.Is(err, playerapi.ErrPlayerNotFound):
		if c.language == LanguageSimplifiedChinese {
			return "未找到该玩家。请确认玩家名和平台。"
		}
		return "Player was not found. Check the player name and platform."
	case errors.Is(err, playerapi.ErrUnknownPlatform):
		if c.language == LanguageSimplifiedChinese {
			return "未知平台。支持 PC、PS4、X1。"
		}
		return "Unknown platform. Supported platforms are PC, PS4, and X1."
	case errors.Is(err, playerapi.ErrRateLimited):
		if c.language == LanguageSimplifiedChinese {
			return "玩家查询达到速率限制，请稍后再试。"
		}
		return "Player lookup is rate limited. Try again later."
	case errors.Is(err, playerapi.ErrPlayerRequired):
		if c.language == LanguageSimplifiedChinese {
			return "请输入玩家名后再查询。"
		}
		return "Enter a player name before searching."
	case errors.Is(err, playerapi.ErrInternalAPI):
		if c.language == LanguageSimplifiedChinese {
			return "玩家查询 API 当前返回内部错误。"
		}
		return "Player lookup API returned an internal error."
	case errors.Is(err, playerapi.ErrExternalAPI):
		if c.language == LanguageSimplifiedChinese {
			return "上游玩家查询暂时不可用。"
		}
		return "Upstream player lookup is temporarily unavailable."
	default:
		if c.language == LanguageSimplifiedChinese {
			return "玩家查询失败。"
		}
		return "Player lookup failed."
	}
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
