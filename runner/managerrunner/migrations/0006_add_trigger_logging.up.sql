-- Migration 0006: Add logging to populate trigger

BEGIN;

-- Replace trigger function with logging version
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
    -- LOG: Trigger started
    RAISE NOTICE 'POPULATE TRIGGER STARTED for result_id=%', NEW.id;

    v_complete_address := NEW.data -> 'complete_address';

    -- Build validation map
    IF NEW.data -> 'email_validations' IS NOT NULL AND jsonb_typeof(NEW.data -> 'email_validations') = 'array' THEN
        FOR v_validation IN SELECT * FROM jsonb_array_elements(NEW.data -> 'email_validations')
        LOOP
            v_validation_map := v_validation_map || jsonb_build_object(lower(trim(v_validation ->> 'email')), v_validation);
        END LOOP;
    END IF;

    -- Insert business_listing
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

    -- LOG: Business listing created
    RAISE NOTICE 'BUSINESS_LISTING created: id=%, title=%', v_listing_id, NEW.data ->> 'title';

    -- Process emails
    IF NEW.data -> 'emails' IS NOT NULL AND jsonb_typeof(NEW.data -> 'emails') = 'array' AND jsonb_array_length(NEW.data -> 'emails') > 0 THEN
        RAISE NOTICE 'PROCESSING EMAILS: count=%', jsonb_array_length(NEW.data -> 'emails');

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

                -- LOG: Email processed
                RAISE NOTICE 'EMAIL processed: % (email_id=%)', v_email, v_email_id;

                v_position := v_position + 1;
            END IF;
        END LOOP;
    ELSE
        RAISE NOTICE 'NO EMAILS in data';
    END IF;

    -- LOG: Trigger completed
    RAISE NOTICE 'POPULATE TRIGGER COMPLETED for result_id=%', NEW.id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMIT;
