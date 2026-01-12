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

export interface DashboardStats {
    total_jobs: number
    active_jobs: number
    completed_jobs: number
    failed_jobs: number
    online_workers: number
    total_results: number
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
    last_updated: string
}

export interface ProxySource {
    id: number
    url: string
    active: boolean
    last_fetch?: string
    status?: 'ok' | 'error'
    error_message?: string
}
