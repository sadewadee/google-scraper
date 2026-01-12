export interface Job {
    id: number
    keyword: string
    status: "pending" | "processing" | "completed" | "failed" | "cancelled"
    created_at: string
    updated_at: string
    result_count?: number
    priority?: "low" | "normal" | "high"
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
