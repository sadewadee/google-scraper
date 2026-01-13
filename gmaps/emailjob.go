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
	// No-reply patterns
	regexp.MustCompile(`^noreply@`),
	regexp.MustCompile(`^no-reply@`),
	regexp.MustCompile(`^donotreply@`),
	regexp.MustCompile(`^do-not-reply@`),
	// UUID-like local parts (32+ hex chars)
	regexp.MustCompile(`^[a-f0-9]{32,}@`),
	// WordPress/CMS generic addresses
	regexp.MustCompile(`@wordpress\.com$`),
	regexp.MustCompile(`^admin@`),
	regexp.MustCompile(`^webmaster@`),
	// Image file extensions (false positives from parsing)
	regexp.MustCompile(`\.(png|jpg|jpeg|gif|svg|webp)$`),
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
		strings.Contains(email, "@test.") {
		return false
	}

	return true
}
