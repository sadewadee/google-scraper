import { useState, useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { useForm } from "react-hook-form"
import { AxiosError } from "axios"
import { toast } from "sonner"
import { jobsApi } from "@/api/jobs"
import type { JobCreatePayload, Job, BoundingBox } from "@/api/types"
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
    Grid,
    RadioGroup,
    Radio,
    FormLabel,
    Typography,
    Chip
} from "@mui/material"
import { Error as ErrorIcon, CheckCircle, GridOn, MyLocation } from "@mui/icons-material"
import { LocationSearch } from "./LocationSearch"

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
    coverage_mode: "single" | "full"
}

interface SelectedLocation {
    name: string
    lat: number
    lon: number
    boundingbox: BoundingBox
}

interface JobFormProps {
    cloneFrom?: Job
    isRetry?: boolean
}

export function JobForm({ cloneFrom, isRetry }: JobFormProps) {
    const navigate = useNavigate()
    const [error, setError] = useState<string | null>(null)
    const [success, setSuccess] = useState(false)
    const [selectedLocation, setSelectedLocation] = useState<SelectedLocation | null>(null)

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
            radius: 5000,
            depth: 10,
            fast_mode: false,
            extract_email: false,
            priority: 5,
            max_time: 600, // 10 minutes
            coverage_mode: "single",
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

    const values = watch()
    const isFastMode = values.fast_mode
    const isExtractEmail = values.extract_email
    const lat = values.lat
    const lon = values.lon
    const coverageMode = values.coverage_mode
    const radius = values.radius

    // Calculate estimated grid points for full coverage mode
    const estimateGridPoints = (): number => {
        if (!selectedLocation?.boundingbox || coverageMode !== "full") return 1

        const bbox = selectedLocation.boundingbox
        const latRange = bbox.max_lat - bbox.min_lat
        const lonRange = bbox.max_lon - bbox.min_lon

        // Convert radius from meters to degrees (approximate)
        const radiusInDegrees = radius / 111320

        const rows = Math.max(1, Math.ceil(latRange / radiusInDegrees))
        const cols = Math.max(1, Math.ceil(lonRange / radiusInDegrees))

        return rows * cols
    }

    const gridPoints = estimateGridPoints()
    const keywordCount = values.keywords.split(/[,\n]/).filter(k => k.trim().length > 0).length
    const totalSearches = gridPoints * Math.max(1, keywordCount)

    // Handle location selection from LocationSearch
    const handleLocationSelect = (location: SelectedLocation) => {
        setSelectedLocation(location)
        setValue("lat", String(location.lat))
        setValue("lon", String(location.lon))
    }

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
                coverage_mode: data.coverage_mode,
            }

            // Add lat/lon if provided
            if (data.lat && data.lon) {
                payload.lat = parseFloat(data.lat)
                payload.lon = parseFloat(data.lon)
            }

            // Add location info if selected
            if (selectedLocation) {
                payload.location_name = selectedLocation.name
                payload.boundingbox = selectedLocation.boundingbox
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
                                        helperText={coverageMode === "full" ? "Also used as grid spacing" : ""}
                                    />
                                </Grid>
                            </Grid>

                            {/* Location Search */}
                            <LocationSearch
                                onLocationSelect={handleLocationSelect}
                            />

                            {/* Coverage Mode - show when location is selected */}
                            {selectedLocation && (
                                <Box sx={{ p: 2, bgcolor: 'grey.50', borderRadius: 2, border: '1px solid', borderColor: 'grey.200' }}>
                                    <FormControl component="fieldset">
                                        <FormLabel component="legend" sx={{ fontWeight: 600, mb: 1 }}>Coverage Mode</FormLabel>
                                        <RadioGroup
                                            row
                                            value={coverageMode}
                                            onChange={(e) => setValue("coverage_mode", e.target.value as "single" | "full")}
                                        >
                                            <FormControlLabel
                                                value="single"
                                                control={<Radio />}
                                                label={
                                                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                                        <MyLocation sx={{ fontSize: 18 }} />
                                                        <Typography variant="body2">Single Point (center only)</Typography>
                                                    </Box>
                                                }
                                            />
                                            <FormControlLabel
                                                value="full"
                                                control={<Radio />}
                                                label={
                                                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                                        <GridOn sx={{ fontSize: 18 }} />
                                                        <Typography variant="body2">Full Coverage (grid search)</Typography>
                                                    </Box>
                                                }
                                            />
                                        </RadioGroup>
                                    </FormControl>

                                    {coverageMode === "full" && (
                                        <Box sx={{ mt: 2, display: 'flex', gap: 2, flexWrap: 'wrap' }}>
                                            <Chip
                                                label={`${gridPoints} grid points`}
                                                color="primary"
                                                variant="outlined"
                                                size="small"
                                            />
                                            <Chip
                                                label={`${keywordCount} keyword${keywordCount !== 1 ? 's' : ''}`}
                                                color="secondary"
                                                variant="outlined"
                                                size="small"
                                            />
                                            <Chip
                                                label={`${totalSearches} total searches`}
                                                color="success"
                                                variant="filled"
                                                size="small"
                                            />
                                        </Box>
                                    )}

                                    {coverageMode === "full" && (
                                        <Alert severity="info" sx={{ mt: 2 }}>
                                            <strong>Full Coverage:</strong> The scraper will search from {gridPoints} points across the entire area.
                                            This provides comprehensive results but takes longer.
                                        </Alert>
                                    )}
                                </Box>
                            )}

                            <Grid container spacing={2}>
                                <Grid size={{ xs: 12, md: 6 }}>
                                    <TextField
                                        label="Latitude"
                                        placeholder="e.g. -6.2088"
                                        fullWidth
                                        required={isFastMode}
                                        error={isFastMode && (!lat || lat.length === 0)}
                                        helperText={isFastMode && (!lat || lat.length === 0) ? "Required for Fast Mode" : "Auto-filled from location search"}
                                        {...register("lat")}
                                        slotProps={{ input: { readOnly: !!selectedLocation } }}
                                    />
                                </Grid>

                                <Grid size={{ xs: 12, md: 6 }}>
                                    <TextField
                                        label="Longitude"
                                        placeholder="e.g. 106.8456"
                                        fullWidth
                                        required={isFastMode}
                                        error={isFastMode && (!lon || lon.length === 0)}
                                        helperText={isFastMode && (!lon || lon.length === 0) ? "Required for Fast Mode" : "Auto-filled from location search"}
                                        {...register("lon")}
                                        slotProps={{ input: { readOnly: !!selectedLocation } }}
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
                    <Alert severity="error" icon={<ErrorIcon sx={{ fontSize: 16 }} />}>
                        {error}
                    </Alert>
                )}

                {success && (
                    <Alert severity="success" icon={<CheckCircle sx={{ fontSize: 16 }} />}>
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
