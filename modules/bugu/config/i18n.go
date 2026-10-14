package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// defaultTagTranslations provides high-accuracy localized translations for common domain tags.
var defaultTagTranslations = map[string]map[string]string{
	"新闻": {
		"en": "news", "ru": "новости", "de": "Nachrichten", "fr": "actualités", "es": "noticias", "ja": "ニュース", "it": "notizie", "ko": "뉴스",
	},
	"技术": {
		"en": "technology", "ru": "технологии", "de": "Technologie", "fr": "technologie", "es": "tecnología", "ja": "技術", "it": "tecnologia", "ko": "기술",
	},
	"科技": {
		"en": "tech", "ru": "технологии", "de": "Tech", "fr": "technologie", "es": "tecnología", "ja": "テクノロジー", "it": "tecnologia", "ko": "테크",
	},
	"编程": {
		"en": "programming", "ru": "программирование", "de": "Programmierung", "fr": "programmation", "es": "programación", "ja": "プログラミング", "it": "programmazione", "ko": "프로그래밍",
	},
	"开发": {
		"en": "development", "ru": "разработка", "de": "Entwicklung", "fr": "développement", "es": "desarrollo", "ja": "開発", "it": "sviluppo", "ko": "개발",
	},
	"开发者": {
		"en": "developers", "ru": "разработчики", "de": "Entwickler", "fr": "développeurs", "es": "desarrolladores", "ja": "開発者", "it": "sviluppatori", "ko": "개발자",
	},
	"极客": {
		"en": "geek", "ru": "гик", "de": "Geek", "fr": "geek", "es": "geek", "ja": "ギーク", "it": "geek", "ko": "긱",
	},
	"黑客": {
		"en": "hacking", "ru": "хакеры", "de": "Hacking", "fr": "hacking", "es": "hacking", "ja": "ハッキング", "it": "hacking", "ko": "해킹",
	},
	"网络安全": {
		"en": "cybersecurity", "ru": "кибербезопасность", "de": "Cybersicherheit", "fr": "cybersécurité", "es": "ciberseguridad", "ja": "サイバーセキュリティ", "it": "sicurezza informatica", "ko": "사이버보안",
	},
	"信息安全": {
		"en": "infosec", "ru": "информационная безопасность", "de": "IT-Sicherheit", "fr": "sécurité informatique", "es": "seguridad informática", "ja": "情報セキュリティ", "it": "sicurezza delle informazioni", "ko": "정보보안",
	},
	"开源": {
		"en": "open-source", "ru": "открытый исходный код", "de": "Open-Source", "fr": "open-source", "es": "código abierto", "ja": "オープンソース", "it": "open source", "ko": "오픈소스",
	},
	"底层": {
		"en": "low-level systems", "ru": "низкоуровневые системы", "de": "Systemnah", "fr": "bas niveau", "es": "bajo nivel", "ja": "低レイヤ", "it": "basso livello", "ko": "로우레벨",
	},
	"系统编程": {
		"en": "systems programming", "ru": "системное программирование", "de": "Systemprogrammierung", "fr": "programmation système", "es": "programación de sistemas", "ja": "システムプログラミング", "it": "programmazione di sistema", "ko": "시스템 프로그래밍",
	},
	"论坛": {
		"en": "forum", "ru": "форум", "de": "Forum", "fr": "forum", "es": "foro", "ja": "フォーラム", "it": "forum", "ko": "포럼",
	},
	"社区": {
		"en": "community", "ru": "сообщество", "de": "Community", "fr": "communauté", "es": "comunidad", "ja": "コミュニティ", "it": "comunità", "ko": "커뮤니티",
	},
	"综合": {
		"en": "general discussion", "ru": "общее", "de": "Allgemeines", "fr": "général", "es": "general", "ja": "総合", "it": "generale", "ko": "종합",
	},
	"数码": {
		"en": "digital gadgets", "ru": "цифровые технологии", "de": "Digital", "fr": "numérique", "es": "digital", "ja": "デジタル", "it": "digitale", "ko": "디지털",
	},
	"硬件": {
		"en": "hardware", "ru": "аппаратное обеспечение", "de": "Hardware", "fr": "matériel", "es": "hardware", "ja": "ハードウェア", "it": "hardware", "ko": "하드웨어",
	},
	"博客": {
		"en": "blog", "ru": "блог", "de": "Blog", "fr": "blog", "es": "blog", "ja": "ブログ", "it": "blog", "ko": "블로그",
	},
	"生活": {
		"en": "lifestyle", "ru": "образ жизни", "de": "Lifestyle", "fr": "mode de vie", "es": "estilo de vida", "ja": "ライフスタイル", "it": "stile di vita", "ko": "라이프스타일",
	},
	"文学": {
		"en": "literature", "ru": "литература", "de": "Literatur", "fr": "littérature", "es": "literatura", "ja": "文学", "it": "letteratura", "ko": "문학",
	},
	"文化": {
		"en": "culture", "ru": "культура", "de": "Kultur", "fr": "culture", "es": "cultura", "ja": "文化", "it": "cultura", "ko": "문화",
	},
	"哲学": {
		"en": "philosophy", "ru": "философия", "de": "Philosophie", "fr": "philosophie", "es": "filosofía", "ja": "哲学", "it": "filosofia", "ko": "철학",
	},
	"人文": {
		"en": "humanities", "ru": "гуманитарные науки", "de": "Geisteswissenschaften", "fr": "sciences humaines", "es": "humanidades", "ja": "人文学", "it": "scienze umane", "ko": "인문학",
	},
	"科学": {
		"en": "science", "ru": "наука", "de": "Wissenschaft", "fr": "science", "es": "ciencia", "ja": "科学", "it": "scienza", "ko": "과학",
	},
	"政治": {
		"en": "politics", "ru": "политика", "de": "Politik", "fr": "politique", "es": "política", "ja": "政治", "it": "politica", "ko": "정치",
	},
	"时政": {
		"en": "current affairs", "ru": "текущие события", "de": "Zeitgeschehen", "fr": "actualité politique", "es": "actualidad", "ja": "時事", "it": "attualità", "ko": "시사",
	},
	"社会": {
		"en": "society", "ru": "общество", "de": "Gesellschaft", "fr": "société", "es": "sociedad", "ja": "社会", "it": "società", "ko": "사회",
	},
	"调查": {
		"en": "investigative journalism", "ru": "расследования", "de": "Recherche", "fr": "investigation", "es": "investigación", "ja": "調査報道", "it": "inchieste", "ko": "탐사보도",
	},
	"独立媒体": {
		"en": "independent media", "ru": "независимые медиа", "de": "Unabhängige Medien", "fr": "médias indépendants", "es": "medios independientes", "ja": "独立系メディア", "it": "media indipendenti", "ko": "독립언론",
	},
	"游戏": {
		"en": "gaming", "ru": "игры", "de": "Gaming", "fr": "jeux vidéo", "es": "videojuegos", "ja": "ゲーム", "it": "videogiochi", "ko": "게임",
	},
	"商业": {
		"en": "business", "ru": "бизнес", "de": "Wirtschaft", "fr": "affaires", "es": "negocios", "ja": "ビジネス", "it": "economia", "ko": "비즈니스",
	},
	"创业": {
		"en": "startups", "ru": "стартапы", "de": "Startups", "fr": "startups", "es": "startups", "ja": "スタートアップ", "it": "startup", "ko": "스타트업",
	},
	"学术": {
		"en": "academia", "ru": "научные исследования", "de": "Akademie", "fr": "académique", "es": "académico", "ja": "学術", "it": "accademia", "ko": "학술",
	},
	"深度": {
		"en": "in-depth", "ru": "глубокий анализ", "de": "Tiefgründig", "fr": "en profondeur", "es": "a fondo", "ja": "深層", "it": "approfondito", "ko": "심층",
	},
	"评测": {
		"en": "reviews", "ru": "обзоры", "de": "Testberichte", "fr": "tests", "es": "reseñas", "ja": "レビュー", "it": "recensioni", "ko": "리뷰",
	},
	"评论": {
		"en": "commentary", "ru": "комментарии", "de": "Kommentare", "fr": "commentaires", "es": "comentarios", "ja": "論評", "it": "commenti", "ko": "평론",
	},
	"问答": {
		"en": "q&a", "ru": "вопросы и ответы", "de": "Fragen & Antworten", "fr": "questions-réponses", "es": "preguntas y respuestas", "ja": "Q&A", "it": "domande e risposte", "ko": "Q&A",
	},
	"教程": {
		"en": "tutorials", "ru": "руководства", "de": "Tutorials", "fr": "tutoriels", "es": "tutoriales", "ja": "チュートリアル", "it": "tutorial", "ko": "튜토리얼",
	},
	"创客": {
		"en": "makers", "ru": "мейкеры", "de": "Maker", "fr": "makers", "es": "makers", "ja": "メイカーズ", "it": "maker", "ko": "메이커",
	},
	"人工智能": {
		"en": "artificial intelligence", "ru": "искусственный интеллект", "de": "Künstliche Intelligenz", "fr": "intelligence artificielle", "es": "inteligencia artificial", "ja": "人工知能", "it": "intelligenza artificiale", "ko": "인공지능",
	},
	"AI": {
		"en": "AI", "ru": "ИИ", "de": "KI", "fr": "IA", "es": "IA", "ja": "AI", "it": "IA", "ko": "AI",
	},
	"IT": {
		"en": "IT", "ru": "ИТ", "de": "IT", "fr": "IT", "es": "TI", "ja": "IT", "it": "IT", "ko": "IT",
	},
	"Linux": {
		"en": "Linux", "ru": "Linux", "de": "Linux", "fr": "Linux", "es": "Linux", "ja": "Linux", "it": "Linux", "ko": "Linux",
	},
	"ACG": {
		"en": "anime/manga", "ru": "аниме/манга", "de": "Anime/Manga", "fr": "anime/manga", "es": "anime/manga", "ja": "アニメ・マンガ", "it": "anime/manga", "ko": "서브컬처",
	},
	"二次元": {
		"en": "acg subculture", "ru": "аниме субкультура", "de": "Anime-Subkultur", "fr": "subculture anime", "es": "subcultura anime", "ja": "サブカルチャー", "it": "subcultura anime", "ko": "서브컬처",
	},
	"暗网": {
		"en": "darknet", "ru": "даркнет", "de": "Darknet", "fr": "darknet", "es": "darknet", "ja": "ダークウェブ", "it": "darknet", "ko": "다크넷",
	},
	"匿名": {
		"en": "anonymous board", "ru": "анонимный борда", "de": "Anonym", "fr": "anonyme", "es": "anónimo", "ja": "匿名掲示板", "it": "anonimo", "ko": "익명",
	},
	"职场": {
		"en": "workplace career", "ru": "карьера", "de": "Karriere", "fr": "carrière", "es": "carrera profesional", "ja": "キャリア", "it": "carriera", "ko": "커리어",
	},
	"闲聊": {
		"en": "casual chat", "ru": "разговоры", "de": "Plauderei", "fr": "bavardage", "es": "charla", "ja": "雑談", "it": "chiacchiere", "ko": "잡담",
	},
	"杂谈": {
		"en": "misc discussion", "ru": "разное", "de": "Vermischtes", "fr": "divers", "es": "miscelánea", "ja": "雑談", "it": "varie", "ko": "잡담",
	},
}

var (
	customMu               sync.RWMutex
	customTagTranslations = make(map[string]map[string]string)
)

// HasTagTranslation checks if a tag has translations in either custom or default dictionary.
func HasTagTranslation(tag string) bool {
	customMu.RLock()
	if _, ok := customTagTranslations[tag]; ok {
		customMu.RUnlock()
		return true
	}
	customMu.RUnlock()

	_, ok := defaultTagTranslations[tag]
	return ok
}

// LoadCustomTranslations reads custom/auto-enriched tag translations from tags_i18n.json.
func LoadCustomTranslations(filePath string) error {
	if filePath == "" {
		return nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var loaded map[string]map[string]string
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("failed to unmarshal custom tags json: %w", err)
	}

	customMu.Lock()
	defer customMu.Unlock()
	for k, v := range loaded {
		customTagTranslations[k] = v
	}
	return nil
}

// MergeCustomTranslations merges newly translated tags and persists them to tags_i18n.json.
func MergeCustomTranslations(newTranslations map[string]map[string]string, filePath string) error {
	if len(newTranslations) == 0 {
		return nil
	}

	customMu.Lock()
	defer customMu.Unlock()

	for k, v := range newTranslations {
		if _, exists := customTagTranslations[k]; !exists {
			customTagTranslations[k] = make(map[string]string)
		}
		for lang, trans := range v {
			customTagTranslations[k][lang] = trans
		}
	}

	if filePath != "" {
		data, err := json.MarshalIndent(customTagTranslations, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal custom tags json: %w", err)
		}
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write custom tags json: %w", err)
		}
	}
	return nil
}

// TranslateTag converts a tag into the target language.
// It checks dynamic custom translations first, then default translations.
func TranslateTag(tag string, targetLang string) string {
	baseLang := strings.ToLower(strings.TrimSpace(targetLang))
	if idx := strings.Index(baseLang, "-"); idx != -1 {
		baseLang = baseLang[:idx]
	}

	// For Chinese targets, maintain native Chinese tags
	if baseLang == "zh" || baseLang == "" {
		return tag
	}

	// 1. Check custom dynamically enriched translations
	customMu.RLock()
	if translations, ok := customTagTranslations[tag]; ok {
		if trans, found := translations[baseLang]; found && trans != "" {
			customMu.RUnlock()
			return trans
		}
		if trans, found := translations["en"]; found && trans != "" {
			customMu.RUnlock()
			return trans
		}
	}
	customMu.RUnlock()

	// 2. Check built-in default translations
	if translations, ok := defaultTagTranslations[tag]; ok {
		if trans, found := translations[baseLang]; found && trans != "" {
			return trans
		}
		// Fallback to English
		if trans, found := translations["en"]; found && trans != "" {
			return trans
		}
	}

	return tag
}

// TranslateTags translates a slice of tags into the target language.
func TranslateTags(tags []string, targetLang string) []string {
	if len(tags) == 0 {
		return nil
	}
	res := make([]string, len(tags))
	for i, t := range tags {
		res[i] = TranslateTag(t, targetLang)
	}
	return res
}
