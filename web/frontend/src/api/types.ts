export interface Job {
    id: string
    name: string
    status: "pending" | "queued" | "running" | "paused" | "completed" | "failed" | "cancelled"
    priority: number
    config: {
        keywords: string[]
        lang: string
        geo_lat?: number
        geo_lon?: number
        zoom: number
        radius: number
        depth: number
        fast_mode: boolean
        extract_email: boolean
        max_time: number
        proxies?: string[]
        location_name?: string
        boundingbox?: BoundingBox
        coverage_mode?: "single" | "full"
        grid_points?: number
    }
    progress: {
        total_places: number
        scraped_places: number
        failed_places: number
        percentage: number
    }
    worker_id?: string
    created_at: string
    updated_at: string
    started_at?: string
    completed_at?: string
    error_message?: string
}

export interface BoundingBox {
    min_lat: number
    max_lat: number
    min_lon: number
    max_lon: number
}

export interface JobCreatePayload {
    name: string
    keywords: string[]
    lang: string
    zoom: number
    radius: number
    depth: number
    fast_mode: boolean
    extract_email: boolean
    priority: number
    max_time: number
    lat?: number
    lon?: number
    location_name?: string
    boundingbox?: BoundingBox
    coverage_mode?: "single" | "full"
}

export interface Worker {
    id: string
    name: string
    status: "online" | "offline" | "busy"
    last_seen: string
    current_job_id?: number | null
    stats: {
        jobs_completed: number
        uptime_seconds: number
    }
}

// Backend Stats structure (matches domain.Stats in Go)
export interface JobStats {
    total: number
    pending: number
    queued: number
    running: number
    paused: number
    completed: number
    failed: number
    cancelled: number
}

export interface WorkerStats {
    total_workers: number
    online_workers: number
    busy_workers: number
    idle_workers: number
}

export interface PlaceStats {
    total_scraped: number
    today: number
    total_emails: number
    rate_per_hour: number
}

export interface DashboardStats {
    jobs: JobStats
    workers: WorkerStats
    places: PlaceStats
}

export interface ApiResponse<T> {
    data: T
    error?: string
    meta?: {
        page: number
        limit: number
        total: number
    }
}

export interface ProxyStats {
    total_proxies: number
    healthy_proxies: number
    dead_proxies?: number
    banned_proxies?: number
    pending_proxies?: number
    avg_uptime?: number
    last_updated: string
}

export interface ProxySource {
    id: number
    url: string
    active: boolean
    last_fetch?: string
    status?: 'ok' | 'error'
    error_message?: string
    created_at?: string
}

export interface Proxy {
    id: number
    ip: string
    port: number
    protocol: string
    country?: string
    uptime?: number
    response_time?: number
    status: 'pending' | 'healthy' | 'dead' | 'banned'
    last_checked?: string
    last_used?: string
    fail_count: number
    success_count: number
    source_url?: string
    created_at: string
    updated_at: string
}

export interface ProxyListResponse {
    data: Proxy[]
    meta: {
        total: number
        page: number
        limit: number
    }
}

export interface Review {
    name: string
    profile_picture: string
    rating: number
    description: string
    images: string[]
    when: string
}

// Result entry from Google Maps scraping (matches gmaps.Entry in Go)
export interface ResultEntry {
    input_id: string
    link: string
    cid: string
    title: string
    categories: string[]
    category: string
    address: string
    open_hours: Record<string, string[]>
    popular_times: Record<string, Record<number, number>>
    web_site: string
    phone: string
    plus_code: string
    review_count: number
    review_rating: number
    reviews_per_rating: Record<number, number>
    latitude: number
    longitude: number
    status: string
    description: string
    reviews_link: string
    thumbnail: string
    timezone: string
    price_range: string
    data_id: string
    place_id: string
    images: { title: string; image: string }[]
    reservations: { link: string; source: string }[]
    order_online: { link: string; source: string }[]
    menu: { link: string; source: string }
    owner: { id: string; name: string; link: string }
    complete_address: {
        borough: string
        street: string
        city: string
        postal_code: string
        state: string
        country: string
    }
    about: {
        id: string
        name: string
        options: { name: string; enabled: boolean }[]
    }[]
    user_reviews: Review[]
    user_reviews_extended: Review[]
    emails: string[]
}

export interface ResultsResponse {
    data: ResultEntry[]
    meta: {
        page: number
        per_page: number
        total: number
        total_pages: number
    }
}
