import axios from "axios"

const baseURL = import.meta.env.VITE_API_URL || "/api/v2"

export const api = axios.create({
    baseURL,
    headers: {
        "Content-Type": "application/json",
    },
})

// Add response interceptor for error handling
api.interceptors.response.use(
    (response) => response,
    (error) => {
        // Handle specific error cases here (e.g., 401 Unauthorized)
        return Promise.reject(error)
    }
)
