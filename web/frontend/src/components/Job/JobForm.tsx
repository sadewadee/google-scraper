import { useState, useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { useForm } from "react-hook-form"
import { AxiosError } from "axios"
import { toast } from "sonner"
import { jobsApi } from "@/api/jobs"
import type { JobCreatePayload, Job } from "@/api/types"
import { AlertCircle, CheckCircle2 } from "lucide-react"
import {
    TextField,
    Button,
    Select,
    MenuItem,
    FormControl,
    InputLabel,
    Checkbox,
    FormControlLabel,
    Box,
    Card as MuiCard,
    CardContent as MuiCardContent,
    CardHeader as MuiCardHeader,
    Alert,
    Stack,
    Grid
} from "@mui/material"

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
    max_time: number
}

interface JobFormProps {
    cloneFrom?: Job
    isRetry?: boolean
}

export function JobForm({ cloneFrom, isRetry }: JobFormProps) {
    const navigate = useNavigate()
    const [error, setError] = useState<string | null>(null)
    const [success, setSuccess] = useState(false)

    const {
        register,
        handleSubmit,
        formState: { isSubmitting, errors },
        reset,
        watch,
        setValue,
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
            max_time: 600, // 10 minutes
        },
    })

    // Pre-fill form when cloning a job
    useEffect(() => {
        if (cloneFrom) {
            setValue("name", isRetry ? `${cloneFrom.name} (Retry)` : `${cloneFrom.name} (Copy)`)
            setValue("keywords", cloneFrom.config.keywords.join("\n"))
            setValue("lang", cloneFrom.config.lang)
            setValue("zoom", cloneFrom.config.zoom)
            setValue("radius", cloneFrom.config.radius)
            setValue("depth", cloneFrom.config.depth)
            setValue("fast_mode", cloneFrom.config.fast_mode)
            setValue("extract_email", cloneFrom.config.extract_email)
            setValue("priority", cloneFrom.priority)
            setValue("max_time", cloneFrom.config.max_time)
            if (cloneFrom.config.geo_lat) setValue("lat", String(cloneFrom.config.geo_lat))
            if (cloneFrom.config.geo_lon) setValue("lon", String(cloneFrom.config.geo_lon))
        }
    }, [cloneFrom, isRetry, setValue])

    const isFastMode = watch("fast_mode")
    const isExtractEmail = watch("extract_email")
    const lat = watch("lat")
    const lon = watch("lon")

    // Check if form is ready for submission (especially for Fast Mode)
    const isReady = !isFastMode || (isFastMode && !!lat && !!lon)

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

        // Validate Fast Mode requirements
        if (data.fast_mode) {
            if (!data.lat || !data.lon) {
                setError("Latitude and Longitude are required for Fast Mode")
                return
            }
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
                max_time: data.max_time,
            }

            // Add lat/lon if provided
            if (data.lat && data.lon) {
                payload.lat = parseFloat(data.lat)
                payload.lon = parseFloat(data.lon)
            }

            const response = await jobsApi.create(payload)

            if (response) {
                setSuccess(true)
                toast.success("Job created successfully")
                reset()
                // Redirect to jobs list after 1 second
                setTimeout(() => {
                    navigate("/jobs")
                }, 1000)
            }
        } catch (err) {
            console.error("Failed to create job:", err)
            if (err instanceof AxiosError) {
                const errorMessage = err.response?.data?.message || "Failed to create job"
                setError(errorMessage)
                toast.error(errorMessage)
            } else {
                setError("An unexpected error occurred")
                toast.error("An unexpected error occurred")
            }
        }
    }

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <Stack spacing={3}>
                <MuiCard>
                    <MuiCardHeader title="Job Details" />
                    <MuiCardContent>
                        <Stack spacing={3}>
                            <TextField
                                label="Job Name"
                                placeholder="e.g. Coffee Shops Jakarta"
                                required
                                fullWidth
                                {...register("name")}
                                error={!!errors.name}
                                helperText={errors.name?.message}
                            />

                            <TextField
                                label="Keywords"
                                placeholder="Enter keywords (one per line or comma-separated)&#10;e.g. coffee shop jakarta&#10;restaurant bandung"
                                multiline
                                rows={4}
                                required
                                fullWidth
                                {...register("keywords")}
                                helperText="Enter search queries for Google Maps. Each keyword will be scraped separately."
                            />

                            <Grid container spacing={2}>
                                <Grid size={{ xs: 12, md: 6 }}>
                                    <FormControl fullWidth>
                                        <InputLabel>Language</InputLabel>
                                        <Select
                                            {...register("lang")}
                                            defaultValue="en"
                                            label="Language"
                                        >
                                            <MenuItem value="en">English</MenuItem>
                                            <MenuItem value="id">Indonesian</MenuItem>
                                            <MenuItem value="de">German</MenuItem>
                                            <MenuItem value="fr">French</MenuItem>
                                            <MenuItem value="es">Spanish</MenuItem>
                                            <MenuItem value="ja">Japanese</MenuItem>
                                            <MenuItem value="ko">Korean</MenuItem>
                                            <MenuItem value="zh">Chinese</MenuItem>
                                        </Select>
                                    </FormControl>
                                </Grid>

                                <Grid size={{ xs: 12, md: 6 }}>
                                    <TextField
                                        label="Priority (0-10)"
                                        type="number"
                                        slotProps={{ htmlInput: { min: 0, max: 10 } }}
                                        fullWidth
                                        {...register("priority", { valueAsNumber: true })}
                                        helperText="Higher = processed first"
                                    />
                                </Grid>
                            </Grid>

                            <TextField
                                label="Max Time (seconds)"
                                type="number"
                                slotProps={{ htmlInput: { min: 180 } }}
                                fullWidth
                                {...register("max_time", { valueAsNumber: true })}
                                helperText="Minimum 180 seconds (3 minutes)"
                            />
                        </Stack>
                    </MuiCardContent>
                </MuiCard>

                <MuiCard>
                    <MuiCardHeader title="Search Settings" />
                    <MuiCardContent>
                        <Stack spacing={3}>
                            <Grid container spacing={2}>
                                <Grid size={{ xs: 12, md: 4 }}>
                                    <TextField
                                        label="Depth (Max Results)"
                                        type="number"
                                        slotProps={{ htmlInput: { min: 1, max: 100 } }}
                                        fullWidth
                                        {...register("depth", { valueAsNumber: true })}
                                    />
                                </Grid>

                                <Grid size={{ xs: 12, md: 4 }}>
                                    <TextField
                                        label="Zoom Level"
                                        type="number"
                                        slotProps={{ htmlInput: { min: 1, max: 21 } }}
                                        fullWidth
                                        {...register("zoom", { valueAsNumber: true })}
                                    />
                                </Grid>

                                <Grid size={{ xs: 12, md: 4 }}>
                                    <TextField
                                        label="Radius (meters)"
                                        type="number"
                                        slotProps={{ htmlInput: { min: 100, max: 50000 } }}
                                        fullWidth
                                        {...register("radius", { valueAsNumber: true })}
                                    />
                                </Grid>
                            </Grid>

                            <Grid container spacing={2}>
                                <Grid size={{ xs: 12, md: 6 }}>
                                    <TextField
                                        label="Latitude"
                                        placeholder="e.g. -6.2088"
                                        fullWidth
                                        required={isFastMode}
                                        error={isFastMode && (!lat || lat.length === 0)}
                                        helperText={isFastMode && (!lat || lat.length === 0) ? "Required for Fast Mode" : ""}
                                        {...register("lat")}
                                    />
                                </Grid>

                                <Grid size={{ xs: 12, md: 6 }}>
                                    <TextField
                                        label="Longitude"
                                        placeholder="e.g. 106.8456"
                                        fullWidth
                                        required={isFastMode}
                                        error={isFastMode && (!lon || lon.length === 0)}
                                        helperText={isFastMode && (!lon || lon.length === 0) ? "Required for Fast Mode" : ""}
                                        {...register("lon")}
                                    />
                                </Grid>
                            </Grid>

                            <Stack spacing={2}>
                                <Box sx={{ display: 'flex', gap: 3 }}>
                                    <FormControlLabel
                                        control={<Checkbox {...register("fast_mode")} />}
                                        label="Fast Mode"
                                    />
                                    <FormControlLabel
                                        control={<Checkbox {...register("extract_email")} />}
                                        label="Extract Emails"
                                    />
                                </Box>

                                {isFastMode && (
                                    <Alert severity="info">
                                        <strong>Fast Mode enabled:</strong> Latitude and Longitude are required. The scraper will simulate a search from that specific location.
                                    </Alert>
                                )}
                                {isExtractEmail && (
                                    <Alert severity="info">
                                        <strong>Email Extraction enabled:</strong> This process takes longer. Ensure Max Time is sufficient.
                                    </Alert>
                                )}
                            </Stack>
                        </Stack>
                    </MuiCardContent>
                </MuiCard>

                {error && (
                    <Alert severity="error" icon={<AlertCircle className="h-4 w-4" />}>
                        {error}
                    </Alert>
                )}

                {success && (
                    <Alert severity="success" icon={<CheckCircle2 className="h-4 w-4" />}>
                        Job created successfully! Redirecting...
                    </Alert>
                )}

                <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 2 }}>
                    <Button
                        variant="outlined"
                        onClick={() => navigate("/jobs")}
                    >
                        Cancel
                    </Button>
                    <Button
                        type="submit"
                        variant="contained"
                        disabled={isSubmitting || !isReady}
                    >
                        {isSubmitting ? "Creating..." : "Create Job"}
                    </Button>
                </Box>
            </Stack>
        </form>
    )
}
