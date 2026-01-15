package gmaps

import (
	"context"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/sadewadee/google-scraper/exiter"
	"github.com/gosom/scrapemate"
	"github.com/mcnijman/go-emailaddress"
)

type EmailExtractJobOptions func(*EmailExtractJob)

type EmailExtractJob struct {
	scrapemate.Job

	Entry       *Entry
	ExitMonitor exiter.Exiter
}

func NewEmailJob(parentID string, entry *Entry, opts ...EmailExtractJobOptions) *EmailExtractJob {
	const (
		defaultPrio       = scrapemate.PriorityHigh
		defaultMaxRetries = 0
	)

	job := EmailExtractJob{
		Job: scrapemate.Job{
			ID:         uuid.New().String(),
			ParentID:   parentID,
			Method:     "GET",
			URL:        entry.WebSite,
			MaxRetries: defaultMaxRetries,
			Priority:   defaultPrio,
		},
	}

	job.Entry = entry

	for _, opt := range opts {
		opt(&job)
	}

	return &job
}

func WithEmailJobExitMonitor(exitMonitor exiter.Exiter) EmailExtractJobOptions {
	return func(j *EmailExtractJob) {
		j.ExitMonitor = exitMonitor
	}
}

func (j *EmailExtractJob) Process(ctx context.Context, resp *scrapemate.Response) (any, []scrapemate.IJob, error) {
	defer func() {
		resp.Document = nil
		resp.Body = nil
	}()

	defer func() {
		if j.ExitMonitor != nil {
			j.ExitMonitor.IncrPlacesCompleted(1)
		}
	}()

	log := scrapemate.GetLoggerFromContext(ctx)

	log.Info("Processing email job", "url", j.URL)

	// if html fetch failed just return
	if resp.Error != nil {
		return j.Entry, nil, nil
	}

	doc, ok := resp.Document.(*goquery.Document)
	if !ok {
		return j.Entry, nil, nil
	}

	emails := docEmailExtractor(doc)
	if len(emails) == 0 {
		emails = regexEmailExtractor(resp.Body)
	}

	// Filter out placeholder/protected emails
	j.Entry.Emails = filterInvalidEmails(emails)

	return j.Entry, nil, nil
}

func (j *EmailExtractJob) ProcessOnFetchError() bool {
	return true
}

func (j *EmailExtractJob) UseInResults() bool {
	return true
}

func docEmailExtractor(doc *goquery.Document) []string {
	seen := map[string]bool{}

	var emails []string

	doc.Find("a[href^='mailto:']").Each(func(_ int, s *goquery.Selection) {
		mailto, exists := s.Attr("href")
		if exists {
			value := strings.TrimPrefix(mailto, "mailto:")
			if email, err := getValidEmail(value); err == nil {
				if !seen[email] {
					emails = append(emails, email)
					seen[email] = true
				}
			}
		}
	})

	return emails
}

func regexEmailExtractor(body []byte) []string {
	seen := map[string]bool{}

	var emails []string

	addresses := emailaddress.Find(body, false)
	for i := range addresses {
		if !seen[addresses[i].String()] {
			emails = append(emails, addresses[i].String())
			seen[addresses[i].String()] = true
		}
	}

	return emails
}

func getValidEmail(s string) (string, error) {
	email, err := emailaddress.Parse(strings.TrimSpace(s))
	if err != nil {
		return "", err
	}

	return email.String(), nil
}

// invalidEmailPatterns contains regex patterns for emails that should be filtered out
var invalidEmailPatterns = []*regexp.Regexp{
	// Wix protection/sentry emails
	regexp.MustCompile(`@sentry\.wixpress\.com$`),
	regexp.MustCompile(`@sentry-next\.wixpress\.com$`),

	// Sentry/Error Tracking Domains
	regexp.MustCompile(`@.*sentry\.io$`),
	regexp.MustCompile(`@.*sentry\..+$`),
	regexp.MustCompile(`@.*bugsnag\.com$`),
	regexp.MustCompile(`@.*rollbar\.com$`),
	regexp.MustCompile(`@.*errortracking\..+$`),

	// Trapmail/Spam Trap/Disposable Domains
	regexp.MustCompile(`@.*trapmail\..+$`),
	regexp.MustCompile(`@.*spamtrap\..+$`),
	regexp.MustCompile(`@.*honeypot\..+$`),
	regexp.MustCompile(`@mailinator\.com$`),
	regexp.MustCompile(`@guerrillamail\..+$`),
	regexp.MustCompile(`@tempmail\..+$`),
	regexp.MustCompile(`@throwaway\..+$`),
	regexp.MustCompile(`@fakeinbox\.com$`),
	regexp.MustCompile(`@temp-mail\..+$`),
	regexp.MustCompile(`@10minutemail\.com$`),
	regexp.MustCompile(`@yopmail\.com$`),
	regexp.MustCompile(`@maildrop\.cc$`),
	regexp.MustCompile(`@dispostable\.com$`),

	// Placeholder/example domains
	regexp.MustCompile(`@example\.(com|org|net)$`),
	regexp.MustCompile(`@domain\.com$`),
	regexp.MustCompile(`@mydomain\.com$`),
	regexp.MustCompile(`@yoursite\.com$`),
	regexp.MustCompile(`@yourcompany\.com$`),
	regexp.MustCompile(`@yourdomain\.com$`),
	regexp.MustCompile(`@sample\.com$`),
	regexp.MustCompile(`@test\.com$`),
	regexp.MustCompile(`@website\.com$`),
	regexp.MustCompile(`@email\.com$`),
	regexp.MustCompile(`@placeholder\..+$`),
	regexp.MustCompile(`@dummy\..+$`),
	regexp.MustCompile(`@fake\..+$`),
	regexp.MustCompile(`@testing\..+$`),
	regexp.MustCompile(`@localhost$`),
	regexp.MustCompile(`@127\.0\.0\.1$`),
	regexp.MustCompile(`@demo\..+$`),
	regexp.MustCompile(`@sitename\..+$`),
	regexp.MustCompile(`@mywebsite\..+$`),
	regexp.MustCompile(`@changeme\..+$`),
	regexp.MustCompile(`@youremail\..+$`),
	regexp.MustCompile(`@your-email\..+$`),
	regexp.MustCompile(`@mail\.com$`),

	// No-reply patterns
	regexp.MustCompile(`^noreply@`),
	regexp.MustCompile(`^no-reply@`),
	regexp.MustCompile(`^donotreply@`),
	regexp.MustCompile(`^do-not-reply@`),

	// UUID-like local parts (32+ hex chars)
	regexp.MustCompile(`^[a-f0-9]{32,}@`),

	// WordPress/CMS generic addresses
	regexp.MustCompile(`@wordpress\.com$`),
	regexp.MustCompile(`@wordpress\.org$`),
	regexp.MustCompile(`@drupal\.org$`),
	regexp.MustCompile(`@joomla\.org$`),
	regexp.MustCompile(`@ghost\.org$`),
	regexp.MustCompile(`@contentful\.com$`),
	regexp.MustCompile(`@strapi\.io$`),
	regexp.MustCompile(`@sanity\.io$`),
	regexp.MustCompile(`@prismic\.io$`),
	regexp.MustCompile(`@netlify\.com$`),
	regexp.MustCompile(`@vercel\.com$`),
	regexp.MustCompile(`^admin@`),
	regexp.MustCompile(`^webmaster@`),

	// Website Builders
	regexp.MustCompile(`@wix\.com$`),
	regexp.MustCompile(`@squarespace\.com$`),
	regexp.MustCompile(`@weebly\.com$`),
	regexp.MustCompile(`@webflow\.com$`),
	regexp.MustCompile(`@duda\.co$`),
	regexp.MustCompile(`@jimdo\.com$`),
	regexp.MustCompile(`@site123\.com$`),
	regexp.MustCompile(`@strikingly\.com$`),
	regexp.MustCompile(`@carrd\.co$`),
	regexp.MustCompile(`@webnode\.com$`),
	regexp.MustCompile(`@zyro\.com$`),
	regexp.MustCompile(`@tilda\.cc$`),
	regexp.MustCompile(`@ucraft\.com$`),

	// E-commerce Platforms
	regexp.MustCompile(`@shopify\.com$`),
	regexp.MustCompile(`@bigcommerce\.com$`),
	regexp.MustCompile(`@woocommerce\.com$`),
	regexp.MustCompile(`@magento\.com$`),
	regexp.MustCompile(`@prestashop\.com$`),
	regexp.MustCompile(`@ecwid\.com$`),
	regexp.MustCompile(`@volusion\.com$`),
	regexp.MustCompile(`@3dcart\.com$`),
	regexp.MustCompile(`@bigcartel\.com$`),
	regexp.MustCompile(`@storenvy\.com$`),

	// Hosting Providers
	regexp.MustCompile(`@godaddy\.com$`),
	regexp.MustCompile(`@hostinger\.com$`),
	regexp.MustCompile(`@bluehost\.com$`),
	regexp.MustCompile(`@hostgator\.com$`),
	regexp.MustCompile(`@namecheap\.com$`),
	regexp.MustCompile(`@siteground\.com$`),
	regexp.MustCompile(`@dreamhost\.com$`),
	regexp.MustCompile(`@ionos\.com$`),
	regexp.MustCompile(`@a2hosting\.com$`),
	regexp.MustCompile(`@inmotionhosting\.com$`),
	regexp.MustCompile(`@hover\.com$`),
	regexp.MustCompile(`@register\.com$`),
	regexp.MustCompile(`@networksolutions\.com$`),
	regexp.MustCompile(`@cloudflare\.com$`),
	regexp.MustCompile(`@digitalocean\.com$`),
	regexp.MustCompile(`@linode\.com$`),
	regexp.MustCompile(`@vultr\.com$`),
	regexp.MustCompile(`@aws\.amazon\.com$`),
	regexp.MustCompile(`@heroku\.com$`),

	// Landing Page Builders
	regexp.MustCompile(`@unbounce\.com$`),
	regexp.MustCompile(`@leadpages\.com$`),
	regexp.MustCompile(`@instapage\.com$`),
	regexp.MustCompile(`@clickfunnels\.com$`),
	regexp.MustCompile(`@landingi\.com$`),

	// Form/Survey Builders
	regexp.MustCompile(`@typeform\.com$`),
	regexp.MustCompile(`@jotform\.com$`),
	regexp.MustCompile(`@wufoo\.com$`),
	regexp.MustCompile(`@formstack\.com$`),
	regexp.MustCompile(`@cognito\.com$`),
	regexp.MustCompile(`@surveymonkey\.com$`),
	regexp.MustCompile(`@google\.com$`),

	// Image file extensions (false positives from parsing)
	regexp.MustCompile(`\.(png|jpg|jpeg|gif|svg|webp)$`),
	regexp.MustCompile(`\.(ico|bmp|tiff|tif|eps|ai|psd|pdf|doc|docx)$`),
	regexp.MustCompile(`\.(mp3|mp4|wav|avi|mov|wmv|flv|mkv)$`),
	regexp.MustCompile(`\.(css|js|json|xml|html|htm|php|asp|aspx)$`),
	regexp.MustCompile(`\.(zip|rar|7z|tar|gz)$`),
}

// filterInvalidEmails removes placeholder, protected, and invalid emails
func filterInvalidEmails(emails []string) []string {
	var valid []string

	for _, email := range emails {
		if isValidBusinessEmail(email) {
			valid = append(valid, email)
		}
	}

	return valid
}

// isValidBusinessEmail checks if an email is a valid business email
func isValidBusinessEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))

	// Check against all invalid patterns
	for _, pattern := range invalidEmailPatterns {
		if pattern.MatchString(email) {
			return false
		}
	}

	// Check for other suspicious patterns
	// Emails with 'sentry', 'placeholder', 'test' in domain
	if strings.Contains(email, "@sentry.") ||
		strings.Contains(email, "placeholder") ||
		strings.Contains(email, "@test.") ||
		strings.Contains(email, "trapmail") ||
		strings.Contains(email, "spamtrap") ||
		strings.Contains(email, "honeypot") ||
		strings.Contains(email, "tempmail") ||
		strings.Contains(email, "disposable") {
		return false
	}

	return true
}
