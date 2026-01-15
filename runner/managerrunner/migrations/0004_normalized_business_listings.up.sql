-- Migration 0004: Normalized Business Listings with Email Validation Tracking
-- (Embedded migration for managerrunner auto-migrate)

BEGIN;

-- Required for trigram text search (fuzzy matching)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ============================================================================
-- 1. BUSINESS_LISTINGS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS business_listings (
    id BIGSERIAL PRIMARY KEY,
    result_id BIGINT NOT NULL REFERENCES results(id) ON DELETE CASCADE,
    job_id UUID REFERENCES jobs_queue(id) ON DELETE SET NULL,
    place_id TEXT,
    cid TEXT,
    data_id TEXT,
    title TEXT NOT NULL,
    category TEXT,
    categories TEXT[],
    address TEXT,
    phone TEXT,
    website TEXT,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    plus_code TEXT,
    timezone TEXT,
    address_street TEXT,
    address_city TEXT,
    address_state TEXT,
    address_postal_code TEXT,
    address_country TEXT,
    review_count INTEGER DEFAULT 0,
    review_rating NUMERIC(3,1),
    status TEXT,
    price_range TEXT,
    description TEXT,
    link TEXT,
    reviews_link TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_business_listings_result_id ON business_listings(result_id);
CREATE INDEX IF NOT EXISTS idx_business_listings_job_id ON business_listings(job_id);
CREATE INDEX IF NOT EXISTS idx_business_listings_place_id ON business_listings(place_id) WHERE place_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_business_listings_cid ON business_listings(cid) WHERE cid IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_business_listings_title_trgm ON business_listings USING gin(title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_business_listings_category ON business_listings(category);
CREATE INDEX IF NOT EXISTS idx_business_listings_address_trgm ON business_listings USING gin(address gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_business_listings_location ON business_listings(latitude, longitude) WHERE latitude IS NOT NULL AND longitude IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_business_listings_review_rating ON business_listings(review_rating DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS idx_business_listings_city ON business_listings(address_city) WHERE address_city IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_business_listings_country ON business_listings(address_country) WHERE address_country IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_business_listings_created_at ON business_listings(created_at DESC);

-- ============================================================================
-- 2. EMAILS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS emails (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    domain TEXT GENERATED ALWAYS AS (
        CASE WHEN position('@' in email) > 0
        THEN lower(substring(email from position('@' in email) + 1))
        ELSE NULL END
    ) STORED,
    local_part TEXT GENERATED ALWAYS AS (
        CASE WHEN position('@' in email) > 0
        THEN lower(substring(email from 1 for position('@' in email) - 1))
        ELSE email END
    ) STORED,
    validation_status TEXT NOT NULL DEFAULT 'pending'
        CHECK (validation_status IN ('pending', 'local_valid', 'local_invalid', 'api_valid', 'api_invalid', 'api_error', 'api_skipped')),
    local_validation_passed BOOLEAN,
    local_validation_reason TEXT,
    local_validated_at TIMESTAMPTZ,
    api_status TEXT,
    api_score NUMERIC(5,2),
    api_deliverable BOOLEAN,
    api_disposable BOOLEAN,
    api_role_account BOOLEAN,
    api_free_email BOOLEAN,
    api_catch_all BOOLEAN,
    api_reason TEXT,
    api_validated_at TIMESTAMPTZ,
    is_acceptable BOOLEAN GENERATED ALWAYS AS (
        CASE
            WHEN validation_status = 'api_valid' THEN true
            WHEN validation_status = 'api_invalid' THEN false
            WHEN validation_status = 'api_error' THEN local_validation_passed
            WHEN validation_status = 'api_skipped' THEN local_validation_passed
            WHEN validation_status = 'local_valid' THEN true
            WHEN validation_status = 'local_invalid' THEN false
            ELSE NULL
        END
    ) STORED,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    occurrence_count INTEGER DEFAULT 1
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_emails_email ON emails(email);
CREATE INDEX IF NOT EXISTS idx_emails_domain ON emails(domain);
CREATE INDEX IF NOT EXISTS idx_emails_validation_status ON emails(validation_status);
CREATE INDEX IF NOT EXISTS idx_emails_is_acceptable ON emails(is_acceptable) WHERE is_acceptable = true;
CREATE INDEX IF NOT EXISTS idx_emails_pending_validation ON emails(first_seen_at ASC) WHERE validation_status = 'pending';
CREATE INDEX IF NOT EXISTS idx_emails_api_score ON emails(api_score DESC NULLS LAST) WHERE api_score IS NOT NULL;

-- ============================================================================
-- 3. BUSINESS_EMAILS JUNCTION TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS business_emails (
    id BIGSERIAL PRIMARY KEY,
    business_listing_id BIGINT NOT NULL REFERENCES business_listings(id) ON DELETE CASCADE,
    email_id BIGINT NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    source TEXT DEFAULT 'website',
    position INTEGER DEFAULT 0,
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(business_listing_id, email_id)
);

CREATE INDEX IF NOT EXISTS idx_business_emails_business_id ON business_emails(business_listing_id);
CREATE INDEX IF NOT EXISTS idx_business_emails_email_id ON business_emails(email_id);

-- ============================================================================
-- 4. TRIGGER FUNCTION
-- ============================================================================

CREATE OR REPLACE FUNCTION populate_normalized_listings()
RETURNS TRIGGER AS $$
DECLARE
    v_listing_id BIGINT;
    v_email TEXT;
    v_email_id BIGINT;
    v_position INTEGER := 0;
    v_complete_address JSONB;
    v_validation JSONB;
    v_validation_map JSONB := '{}'::JSONB;
BEGIN
    v_complete_address := NEW.data -> 'complete_address';

    -- Build a map of email -> validation metadata from email_validations array
    IF NEW.data -> 'email_validations' IS NOT NULL AND jsonb_typeof(NEW.data -> 'email_validations') = 'array' THEN
        FOR v_validation IN SELECT * FROM jsonb_array_elements(NEW.data -> 'email_validations')
        LOOP
            v_validation_map := v_validation_map || jsonb_build_object(lower(trim(v_validation ->> 'email')), v_validation);
        END LOOP;
    END IF;

    INSERT INTO business_listings (
        result_id, job_id, place_id, cid, data_id, title, category, categories,
        address, phone, website, latitude, longitude, plus_code, timezone,
        address_street, address_city, address_state, address_postal_code, address_country,
        review_count, review_rating, status, price_range, description, link, reviews_link
    ) VALUES (
        NEW.id, NEW.job_id, NEW.data ->> 'place_id', NEW.data ->> 'cid', NEW.data ->> 'data_id',
        COALESCE(NEW.data ->> 'title', 'Unknown'), NEW.data ->> 'category',
        CASE WHEN NEW.data -> 'categories' IS NOT NULL AND jsonb_typeof(NEW.data -> 'categories') = 'array'
        THEN ARRAY(SELECT jsonb_array_elements_text(NEW.data -> 'categories')) ELSE NULL END,
        NEW.data ->> 'address', NEW.data ->> 'phone', NEW.data ->> 'web_site',
        (NEW.data ->> 'latitude')::DOUBLE PRECISION, (NEW.data ->> 'longitude')::DOUBLE PRECISION,
        NEW.data ->> 'plus_code', NEW.data ->> 'timezone',
        v_complete_address ->> 'street', v_complete_address ->> 'city',
        v_complete_address ->> 'state', v_complete_address ->> 'postal_code', v_complete_address ->> 'country',
        COALESCE((NEW.data ->> 'review_count')::INTEGER, 0), (NEW.data ->> 'review_rating')::NUMERIC(3,1),
        NEW.data ->> 'status', NEW.data ->> 'price_range', NEW.data ->> 'description',
        NEW.data ->> 'link', NEW.data ->> 'reviews_link'
    )
    ON CONFLICT (result_id) DO UPDATE SET
        job_id = EXCLUDED.job_id, title = EXCLUDED.title, category = EXCLUDED.category,
        address = EXCLUDED.address, phone = EXCLUDED.phone, website = EXCLUDED.website,
        review_count = EXCLUDED.review_count, review_rating = EXCLUDED.review_rating,
        status = EXCLUDED.status, updated_at = NOW()
    RETURNING id INTO v_listing_id;

    IF NEW.data -> 'emails' IS NOT NULL AND jsonb_typeof(NEW.data -> 'emails') = 'array' AND jsonb_array_length(NEW.data -> 'emails') > 0 THEN
        FOR v_email IN SELECT jsonb_array_elements_text(NEW.data -> 'emails')
        LOOP
            v_email := lower(trim(v_email));
            IF v_email IS NOT NULL AND v_email != '' THEN
                v_validation := v_validation_map -> v_email;

                IF v_validation IS NOT NULL THEN
                    INSERT INTO emails (email, validation_status, local_validation_passed, local_validated_at,
                        api_status, api_score, api_deliverable, api_disposable, api_role_account,
                        api_free_email, api_catch_all, api_reason, api_validated_at)
                    VALUES (v_email,
                        CASE
                            WHEN (v_validation ->> 'status') = 'api_error' THEN 'api_error'
                            WHEN (v_validation ->> 'status') = 'valid'
                                 AND (v_validation ->> 'deliverable')::BOOLEAN = true
                                 AND (v_validation ->> 'disposable')::BOOLEAN = false
                                 AND (v_validation ->> 'role_account')::BOOLEAN = false
                                 AND COALESCE((v_validation ->> 'score')::NUMERIC, 0) >= 70
                            THEN 'api_valid'
                            ELSE 'api_invalid'
                        END,
                        true, NOW(),
                        v_validation ->> 'status', (v_validation ->> 'score')::NUMERIC,
                        (v_validation ->> 'deliverable')::BOOLEAN, (v_validation ->> 'disposable')::BOOLEAN,
                        (v_validation ->> 'role_account')::BOOLEAN, (v_validation ->> 'free_email')::BOOLEAN,
                        (v_validation ->> 'catch_all')::BOOLEAN, v_validation ->> 'reason', NOW())
                    ON CONFLICT (email) DO UPDATE SET
                        last_seen_at = NOW(), occurrence_count = emails.occurrence_count + 1,
                        validation_status = CASE WHEN EXCLUDED.api_validated_at IS NOT NULL THEN EXCLUDED.validation_status ELSE emails.validation_status END,
                        api_status = COALESCE(EXCLUDED.api_status, emails.api_status),
                        api_score = COALESCE(EXCLUDED.api_score, emails.api_score),
                        api_deliverable = COALESCE(EXCLUDED.api_deliverable, emails.api_deliverable),
                        api_disposable = COALESCE(EXCLUDED.api_disposable, emails.api_disposable),
                        api_role_account = COALESCE(EXCLUDED.api_role_account, emails.api_role_account),
                        api_free_email = COALESCE(EXCLUDED.api_free_email, emails.api_free_email),
                        api_catch_all = COALESCE(EXCLUDED.api_catch_all, emails.api_catch_all),
                        api_reason = COALESCE(EXCLUDED.api_reason, emails.api_reason),
                        api_validated_at = COALESCE(EXCLUDED.api_validated_at, emails.api_validated_at)
                    RETURNING id INTO v_email_id;
                ELSE
                    INSERT INTO emails (email, validation_status, local_validation_passed, local_validated_at)
                    VALUES (v_email, 'local_valid', true, NOW())
                    ON CONFLICT (email) DO UPDATE SET last_seen_at = NOW(), occurrence_count = emails.occurrence_count + 1
                    RETURNING id INTO v_email_id;
                END IF;

                INSERT INTO business_emails (business_listing_id, email_id, position, source)
                VALUES (v_listing_id, v_email_id, v_position, 'website')
                ON CONFLICT (business_listing_id, email_id) DO NOTHING;

                v_position := v_position + 1;
            END IF;
        END LOOP;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_populate_normalized_listings ON results;
CREATE TRIGGER trg_populate_normalized_listings
    AFTER INSERT ON results
    FOR EACH ROW
    EXECUTE FUNCTION populate_normalized_listings();

-- ============================================================================
-- 5. HELPER FUNCTIONS
-- ============================================================================

CREATE OR REPLACE FUNCTION update_email_validation(
    p_email TEXT, p_status TEXT, p_score NUMERIC, p_deliverable BOOLEAN,
    p_disposable BOOLEAN, p_role_account BOOLEAN, p_free_email BOOLEAN,
    p_catch_all BOOLEAN, p_reason TEXT
) RETURNS BOOLEAN AS $$
DECLARE
    v_validation_status TEXT;
BEGIN
    IF p_status = 'valid' AND p_deliverable = true AND p_disposable = false AND p_role_account = false AND p_score >= 70 THEN
        v_validation_status := 'api_valid';
    ELSE
        v_validation_status := 'api_invalid';
    END IF;

    UPDATE emails SET
        validation_status = v_validation_status, api_status = p_status, api_score = p_score,
        api_deliverable = p_deliverable, api_disposable = p_disposable, api_role_account = p_role_account,
        api_free_email = p_free_email, api_catch_all = p_catch_all, api_reason = p_reason, api_validated_at = NOW()
    WHERE email = lower(trim(p_email));
    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION mark_email_validation_error(p_email TEXT, p_reason TEXT) RETURNS BOOLEAN AS $$
BEGIN
    UPDATE emails SET validation_status = 'api_error', api_reason = p_reason, api_validated_at = NOW()
    WHERE email = lower(trim(p_email));
    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 6. VIEWS
-- ============================================================================

CREATE OR REPLACE VIEW v_business_listings_with_emails AS
SELECT
    bl.id, bl.result_id, bl.job_id, bl.place_id, bl.cid, bl.title, bl.category, bl.categories,
    bl.address, bl.phone, bl.website, bl.latitude, bl.longitude, bl.review_count, bl.review_rating,
    bl.status, bl.price_range, bl.address_city, bl.address_country, bl.link, bl.created_at,
    COALESCE(jsonb_agg(DISTINCT jsonb_build_object('email', e.email, 'status', e.validation_status, 'score', e.api_score, 'is_acceptable', e.is_acceptable)) FILTER (WHERE e.id IS NOT NULL), '[]'::jsonb) AS emails_with_validation,
    COALESCE(array_agg(DISTINCT e.email ORDER BY e.email) FILTER (WHERE e.id IS NOT NULL), ARRAY[]::TEXT[]) AS emails,
    COUNT(DISTINCT e.id) FILTER (WHERE e.is_acceptable = true) AS valid_email_count,
    COUNT(DISTINCT e.id) AS total_email_count
FROM business_listings bl
LEFT JOIN business_emails be ON be.business_listing_id = bl.id
LEFT JOIN emails e ON e.id = be.email_id
GROUP BY bl.id;

CREATE OR REPLACE VIEW v_email_validation_queue AS
SELECT e.id, e.email, e.domain, e.validation_status, e.first_seen_at, e.occurrence_count,
    ROW_NUMBER() OVER (ORDER BY e.occurrence_count DESC, e.first_seen_at ASC) as priority
FROM emails e WHERE e.validation_status IN ('pending', 'local_valid');

-- ============================================================================
-- 7. BACKFILL FUNCTION
-- ============================================================================

CREATE OR REPLACE FUNCTION backfill_normalized_listings(p_batch_size INTEGER DEFAULT 1000)
RETURNS TABLE(processed INTEGER, errors INTEGER) AS $$
DECLARE
    v_processed INTEGER := 0; v_errors INTEGER := 0; v_result RECORD;
    v_listing_id BIGINT; v_email TEXT; v_email_id BIGINT; v_position INTEGER; v_complete_address JSONB;
BEGIN
    FOR v_result IN SELECT r.* FROM results r LEFT JOIN business_listings bl ON bl.result_id = r.id WHERE bl.id IS NULL ORDER BY r.id LIMIT p_batch_size
    LOOP
        BEGIN
            v_complete_address := v_result.data -> 'complete_address'; v_position := 0;
            INSERT INTO business_listings (result_id, job_id, place_id, cid, data_id, title, category, categories, address, phone, website, latitude, longitude, plus_code, timezone, address_street, address_city, address_state, address_postal_code, address_country, review_count, review_rating, status, price_range, description, link, reviews_link)
            VALUES (v_result.id, v_result.job_id, v_result.data ->> 'place_id', v_result.data ->> 'cid', v_result.data ->> 'data_id', COALESCE(v_result.data ->> 'title', 'Unknown'), v_result.data ->> 'category', CASE WHEN v_result.data -> 'categories' IS NOT NULL AND jsonb_typeof(v_result.data -> 'categories') = 'array' THEN ARRAY(SELECT jsonb_array_elements_text(v_result.data -> 'categories')) ELSE NULL END, v_result.data ->> 'address', v_result.data ->> 'phone', v_result.data ->> 'web_site', (v_result.data ->> 'latitude')::DOUBLE PRECISION, (v_result.data ->> 'longitude')::DOUBLE PRECISION, v_result.data ->> 'plus_code', v_result.data ->> 'timezone', v_complete_address ->> 'street', v_complete_address ->> 'city', v_complete_address ->> 'state', v_complete_address ->> 'postal_code', v_complete_address ->> 'country', COALESCE((v_result.data ->> 'review_count')::INTEGER, 0), (v_result.data ->> 'review_rating')::NUMERIC(3,1), v_result.data ->> 'status', v_result.data ->> 'price_range', v_result.data ->> 'description', v_result.data ->> 'link', v_result.data ->> 'reviews_link')
            ON CONFLICT (result_id) DO NOTHING RETURNING id INTO v_listing_id;

            IF v_listing_id IS NOT NULL AND v_result.data -> 'emails' IS NOT NULL AND jsonb_typeof(v_result.data -> 'emails') = 'array' THEN
                FOR v_email IN SELECT jsonb_array_elements_text(v_result.data -> 'emails')
                LOOP
                    v_email := lower(trim(v_email));
                    IF v_email IS NOT NULL AND v_email != '' THEN
                        INSERT INTO emails (email, validation_status, local_validation_passed, local_validated_at) VALUES (v_email, 'local_valid', true, NOW()) ON CONFLICT (email) DO UPDATE SET last_seen_at = NOW(), occurrence_count = emails.occurrence_count + 1 RETURNING id INTO v_email_id;
                        INSERT INTO business_emails (business_listing_id, email_id, position, source) VALUES (v_listing_id, v_email_id, v_position, 'website') ON CONFLICT (business_listing_id, email_id) DO NOTHING;
                        v_position := v_position + 1;
                    END IF;
                END LOOP;
            END IF;
            v_processed := v_processed + 1;
        EXCEPTION WHEN OTHERS THEN
            v_errors := v_errors + 1;
            RAISE WARNING 'Error processing result %: %', v_result.id, SQLERRM;
        END;
    END LOOP;
    RETURN QUERY SELECT v_processed, v_errors;
END;
$$ LANGUAGE plpgsql;

COMMIT;
