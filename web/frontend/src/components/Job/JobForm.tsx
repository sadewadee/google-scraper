import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { useForm } from "react-hook-form"
import { AxiosError } from "axios"
import { Button } from "@/components/UI/Button"
import { Input } from "@/components/UI/Input"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/UI/Card"
import { jobsApi } from "@/api/jobs"
import type { JobCreatePayload } from "@/api/types"
import { AlertCircle, CheckCircle2 } from "lucide-react"

interface FormData {
    name: string
    keywords: string
    lang: string
    lat: string
    lon: string
    zoom: number
    radius: number
    depth: number
    fast_mode: boolean
    extract_email: boolean
    priority: number
}

export function JobForm() {
    const navigate = useNavigate()
    const [error, setError] = useState<string | null>(null)
    const [success, setSuccess] = useState(false)

    const {
        register,
        handleSubmit,
        formState: { isSubmitting },
        reset,
    } = useForm<FormData>({
        defaultValues: {
            name: "",
            keywords: "",
            lang: "en",
            lat: "",
            lon: "",
            zoom: 15,
            radius: 10000,
            depth: 10,
            fast_mode: false,
            extract_email: false,
            priority: 5,
        },
    })

    async function onSubmit(data: FormData) {
        setError(null)
        setSuccess(false)

        // Validate required fields
        if (!data.name || data.name.trim().length === 0) {
            setError("Job name is required")
            return
        }

        if (!data.keywords || data.keywords.trim().length < 3) {
            setError("Enter at least one keyword")
            return
        }

        try {
            // Parse keywords from comma-separated or newline-separated string
            const keywords = data.keywords
                .split(/[,\n]/)
                .map((k) => k.trim())
                .filter((k) => k.length > 0)

            if (keywords.length === 0) {
                setError("At least one keyword is required")
                return
            }

            // Build request payload matching backend API
            const payload: JobCreatePayload = {
                name: data.name,
                keywords: keywords,
                lang: data.lang,
                zoom: data.zoom,
                radius: data.radius,
                depth: data.depth,
                fast_mode: data.fast_mode,
                extract_email: data.extract_email,
                priority: data.priority,
            }

            // Add lat/lon if provided
            if (data.lat && data.lon) {
                payload.lat = parseFloat(data.lat)
                payload.lon = parseFloat(data.lon)
            }

            const response = await jobsApi.create(payload)

            if (response) {
                setSuccess(true)
                reset()
                // Redirect to jobs list after 1 second
                setTimeout(() => {
                    navigate("/jobs")
                }, 1000)
            }
        } catch (err) {
            console.error("Failed to create job:", err)
            if (err instanceof AxiosError) {
                setError(err.response?.data?.message || "Failed to create job")
            } else {
                setError("An unexpected error occurred")
            }
        }
    }

    return (
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
            <Card>
                <CardHeader>
                    <CardTitle>Job Details</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="space-y-2">
                        <label className="text-sm font-medium">Job Name *</label>
                        <Input
                            placeholder="e.g. Coffee Shops Jakarta"
                            {...register("name")}
                        />
                    </div>

                    <div className="space-y-2">
                        <label className="text-sm font-medium">Keywords *</label>
                        <textarea
                            placeholder="Enter keywords (one per line or comma-separated)&#10;e.g. coffee shop jakarta&#10;restaurant bandung"
                            {...register("keywords")}
                            rows={4}
                            className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                        />
                        <p className="text-xs text-muted-foreground">
                            Enter search queries for Google Maps. Each keyword will be scraped separately.
                        </p>
                    </div>

                    <div className="grid gap-4 md:grid-cols-2">
                        <div className="space-y-2">
                            <label className="text-sm font-medium">Language</label>
                            <select
                                {...register("lang")}
                                className="flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
                            >
                                <option value="en">English</option>
                                <option value="id">Indonesian</option>
                                <option value="de">German</option>
                                <option value="fr">French</option>
                                <option value="es">Spanish</option>
                                <option value="ja">Japanese</option>
                                <option value="ko">Korean</option>
                                <option value="zh">Chinese</option>
                            </select>
                        </div>

                        <div className="space-y-2">
                            <label className="text-sm font-medium">Priority (0-10)</label>
                            <Input
                                type="number"
                                min={0}
                                max={10}
                                {...register("priority", { valueAsNumber: true })}
                            />
                            <p className="text-xs text-muted-foreground">Higher = processed first</p>
                        </div>
                    </div>
                </CardContent>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle>Search Settings</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="grid gap-4 md:grid-cols-3">
                        <div className="space-y-2">
                            <label className="text-sm font-medium">Depth (Max Results)</label>
                            <Input
                                type="number"
                                min={1}
                                max={100}
                                {...register("depth", { valueAsNumber: true })}
                            />
                        </div>

                        <div className="space-y-2">
                            <label className="text-sm font-medium">Zoom Level</label>
                            <Input
                                type="number"
                                min={1}
                                max={21}
                                {...register("zoom", { valueAsNumber: true })}
                            />
                        </div>

                        <div className="space-y-2">
                            <label className="text-sm font-medium">Radius (meters)</label>
                            <Input
                                type="number"
                                min={100}
                                max={50000}
                                {...register("radius", { valueAsNumber: true })}
                            />
                        </div>
                    </div>

                    <div className="grid gap-4 md:grid-cols-2">
                        <div className="space-y-2">
                            <label className="text-sm font-medium">Latitude (Optional)</label>
                            <Input
                                placeholder="e.g. -6.2088"
                                {...register("lat")}
                            />
                        </div>

                        <div className="space-y-2">
                            <label className="text-sm font-medium">Longitude (Optional)</label>
                            <Input
                                placeholder="e.g. 106.8456"
                                {...register("lon")}
                            />
                        </div>
                    </div>

                    <div className="flex gap-6">
                        <label className="flex items-center gap-2 cursor-pointer">
                            <input
                                type="checkbox"
                                {...register("fast_mode")}
                                className="h-4 w-4 rounded border-gray-300"
                            />
                            <span className="text-sm">Fast Mode</span>
                        </label>

                        <label className="flex items-center gap-2 cursor-pointer">
                            <input
                                type="checkbox"
                                {...register("extract_email")}
                                className="h-4 w-4 rounded border-gray-300"
                            />
                            <span className="text-sm">Extract Emails</span>
                        </label>
                    </div>
                </CardContent>
            </Card>

            {error && (
                <div className="flex items-center gap-2 p-3 rounded-md bg-destructive/10 text-destructive">
                    <AlertCircle className="h-4 w-4" />
                    {error}
                </div>
            )}

            {success && (
                <div className="flex items-center gap-2 p-3 rounded-md bg-green-500/10 text-green-600">
                    <CheckCircle2 className="h-4 w-4" />
                    Job created successfully! Redirecting...
                </div>
            )}

            <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => navigate("/jobs")}>
                    Cancel
                </Button>
                <Button type="submit" disabled={isSubmitting}>
                    {isSubmitting ? "Creating..." : "Create Job"}
                </Button>
            </div>
        </form>
    )
}
