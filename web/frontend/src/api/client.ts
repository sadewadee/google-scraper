import axios from "axios"

const baseURL = import.meta.env.VITE_API_URL || "/api/v2"

export const api = axios.create({
    baseURL,
    headers: {
        "Content-Type": "application/json",
    },
})

// Get API key from localStorage
export const getApiKey = (): string | null => {
    return localStorage.getItem("api_key")
}

// Set API key to localStorage
export const setApiKey = (key: string): void => {
    localStorage.setItem("api_key", key)
}

// Remove API key from localStorage
export const removeApiKey = (): void => {
    localStorage.removeItem("api_key")
}

// Check if user is authenticated
export const isAuthenticated = (): boolean => {
    return !!getApiKey()
}

// Add request interceptor to include API key
api.interceptors.request.use(
    (config) => {
        const apiKey = getApiKey()
        if (apiKey) {
            config.headers.Authorization = `Bearer ${apiKey}`
        }
        return config
    },
    (error) => {
        return Promise.reject(error)
    }
)

// Add response interceptor for error handling
api.interceptors.response.use(
    (response) => response,
    (error) => {
        // Handle 401 Unauthorized - redirect to login
        if (error.response?.status === 401) {
            removeApiKey()
            window.location.href = "/login"
        }
        return Promise.reject(error)
    }
)
