import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { api, setApiKey } from "@/api/client"
import { Button } from "@/components/UI/Button"
import { Input } from "@/components/UI/Input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/UI/Card"
import { KeyRound, AlertCircle } from "lucide-react"

export default function Login() {
    const [apiKeyInput, setApiKeyInput] = useState("")
    const [error, setError] = useState("")
    const [loading, setLoading] = useState(false)
    const navigate = useNavigate()

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        setError("")
        setLoading(true)

        if (!apiKeyInput.trim()) {
            setError("API Key is required")
            setLoading(false)
            return
        }

        try {
            // Test the API key by making a request to stats endpoint
            const response = await api.get("/stats", {
                headers: {
                    Authorization: `Bearer ${apiKeyInput}`,
                },
            })

            if (response.status === 200) {
                setApiKey(apiKeyInput)
                navigate("/")
            }
        } catch (err: any) {
            if (err.response?.status === 401) {
                setError("Invalid API Key")
            } else {
                setError("Failed to connect to server")
            }
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="min-h-screen flex items-center justify-center bg-background p-4">
            <Card className="w-full max-w-md">
                <CardHeader className="text-center">
                    <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
                        <KeyRound className="h-6 w-6 text-primary" />
                    </div>
                    <CardTitle className="text-2xl">Scrapy Kremlit</CardTitle>
                    <CardDescription>Enter your API Key to access the dashboard</CardDescription>
                </CardHeader>
                <CardContent>
                    <form onSubmit={handleSubmit} className="space-y-4">
                        <div className="space-y-2">
                            <label className="text-sm font-medium leading-none">API Key</label>
                            <Input
                                type="password"
                                placeholder="Enter your API key"
                                value={apiKeyInput}
                                onChange={(e) => setApiKeyInput(e.target.value)}
                                autoFocus
                            />
                        </div>

                        {error && (
                            <div className="flex items-center gap-2 text-sm text-destructive">
                                <AlertCircle className="h-4 w-4" />
                                {error}
                            </div>
                        )}

                        <Button type="submit" className="w-full" disabled={loading}>
                            {loading ? "Verifying..." : "Login"}
                        </Button>
                    </form>
                </CardContent>
            </Card>
        </div>
    )
}
