import { useState, useEffect, useCallback } from "react"
import {
    TextField,
    Autocomplete,
    Box,
    Typography,
    CircularProgress,
    InputAdornment
} from "@mui/material"
import { LocationOn, Search } from "@mui/icons-material"

export interface NominatimResult {
    place_id: number
    display_name: string
    lat: string
    lon: string
    boundingbox: [string, string, string, string] // [min_lat, max_lat, min_lon, max_lon]
    type: string
    class: string
}

export interface BoundingBox {
    min_lat: number
    max_lat: number
    min_lon: number
    max_lon: number
}

interface LocationSearchProps {
    onLocationSelect: (location: {
        name: string
        lat: number
        lon: number
        boundingbox: BoundingBox
    }) => void
    initialValue?: string
}

// Debounce helper
function useDebounce<T>(value: T, delay: number): T {
    const [debouncedValue, setDebouncedValue] = useState<T>(value)

    useEffect(() => {
        const handler = setTimeout(() => {
            setDebouncedValue(value)
        }, delay)

        return () => {
            clearTimeout(handler)
        }
    }, [value, delay])

    return debouncedValue
}

export function LocationSearch({ onLocationSelect, initialValue = "" }: LocationSearchProps) {
    const [inputValue, setInputValue] = useState(initialValue)
    const [options, setOptions] = useState<NominatimResult[]>([])
    const [loading, setLoading] = useState(false)
    const [selectedLocation, setSelectedLocation] = useState<NominatimResult | null>(null)

    const debouncedInput = useDebounce(inputValue, 1000) // 1 second debounce per Nominatim usage policy

    // Search Nominatim API
    const searchLocation = useCallback(async (query: string) => {
        if (!query || query.length < 3) {
            setOptions([])
            return
        }

        setLoading(true)
        try {
            const response = await fetch(
                `https://nominatim.openstreetmap.org/search?` +
                `q=${encodeURIComponent(query)}&format=json&limit=5&addressdetails=1`,
                {
                    headers: {
                        'Accept': 'application/json',
                        'User-Agent': 'GoogleMapsScraper/1.0'
                    }
                }
            )

            if (!response.ok) {
                throw new Error('Failed to fetch locations')
            }

            const data: NominatimResult[] = await response.json()
            setOptions(data)
        } catch (error) {
            console.error("Location search error:", error)
            setOptions([])
        } finally {
            setLoading(false)
        }
    }, [])

    // Search when debounced input changes
    useEffect(() => {
        if (debouncedInput && !selectedLocation) {
            searchLocation(debouncedInput)
        }
    }, [debouncedInput, searchLocation, selectedLocation])

    const handleSelect = (_event: React.SyntheticEvent, value: NominatimResult | null) => {
        setSelectedLocation(value)

        if (value) {
            const lat = parseFloat(value.lat)
            const lon = parseFloat(value.lon)
            const minLat = parseFloat(value.boundingbox[0])
            const maxLat = parseFloat(value.boundingbox[1])
            const minLon = parseFloat(value.boundingbox[2])
            const maxLon = parseFloat(value.boundingbox[3])

            // Validate parsed values
            if (isNaN(lat) || isNaN(lon) || isNaN(minLat) || isNaN(maxLat) || isNaN(minLon) || isNaN(maxLon)) {
                console.error("Invalid coordinates from Nominatim:", value)
                return
            }

            const boundingbox: BoundingBox = {
                min_lat: minLat,
                max_lat: maxLat,
                min_lon: minLon,
                max_lon: maxLon
            }

            onLocationSelect({
                name: value.display_name,
                lat,
                lon,
                boundingbox
            })
        }
    }

    // Calculate area size for display
    const calculateAreaSize = (bbox: BoundingBox): string => {
        // Approximate calculation (1 degree ≈ 111km at equator)
        const latDiff = bbox.max_lat - bbox.min_lat
        const lonDiff = bbox.max_lon - bbox.min_lon
        const avgLat = (bbox.max_lat + bbox.min_lat) / 2

        const heightKm = latDiff * 111
        const widthKm = lonDiff * 111 * Math.cos(avgLat * Math.PI / 180)

        return `~${Math.round(widthKm)}km x ${Math.round(heightKm)}km`
    }

    return (
        <Autocomplete
            fullWidth
            freeSolo
            options={options}
            loading={loading}
            inputValue={inputValue}
            onInputChange={(_event, newInputValue) => {
                setInputValue(newInputValue)
                if (!newInputValue) {
                    setSelectedLocation(null)
                }
            }}
            onChange={handleSelect}
            getOptionLabel={(option) =>
                typeof option === 'string' ? option : option.display_name
            }
            isOptionEqualToValue={(option, value) => option.place_id === value.place_id}
            filterOptions={(x) => x} // Don't filter, use API results directly
            renderOption={(props, option) => {
                const { key, ...rest } = props
                const minLat = parseFloat(option.boundingbox[0])
                const maxLat = parseFloat(option.boundingbox[1])
                const minLon = parseFloat(option.boundingbox[2])
                const maxLon = parseFloat(option.boundingbox[3])

                // Skip invalid bounding boxes
                if (isNaN(minLat) || isNaN(maxLat) || isNaN(minLon) || isNaN(maxLon)) {
                    return (
                        <Box component="li" key={key} {...rest} sx={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-start !important', py: 1.5 }}>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                <LocationOn sx={{ fontSize: 18, color: 'text.secondary' }} />
                                <Typography variant="body2" sx={{ fontWeight: 500 }}>
                                    {option.display_name}
                                </Typography>
                            </Box>
                        </Box>
                    )
                }

                const bbox: BoundingBox = {
                    min_lat: minLat,
                    max_lat: maxLat,
                    min_lon: minLon,
                    max_lon: maxLon
                }

                return (
                    <Box component="li" key={key} {...rest} sx={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-start !important', py: 1.5 }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            <LocationOn sx={{ fontSize: 18, color: 'text.secondary' }} />
                            <Typography variant="body2" sx={{ fontWeight: 500 }}>
                                {option.display_name}
                            </Typography>
                        </Box>
                        <Typography variant="caption" sx={{ color: 'text.secondary', ml: 3.5 }}>
                            {option.type} • {calculateAreaSize(bbox)}
                        </Typography>
                    </Box>
                )
            }}
            renderInput={(params) => (
                <TextField
                    {...params}
                    label="Location"
                    placeholder="Search city, province, country..."
                    helperText="Search for a location to auto-fill coordinates"
                    slotProps={{
                        input: {
                            ...params.InputProps,
                            startAdornment: (
                                <InputAdornment position="start">
                                    <Search sx={{ fontSize: 20, color: 'text.secondary' }} />
                                </InputAdornment>
                            ),
                            endAdornment: (
                                <>
                                    {loading ? <CircularProgress color="inherit" size={20} /> : null}
                                    {params.InputProps.endAdornment}
                                </>
                            ),
                        }
                    }}
                />
            )}
        />
    )
}
