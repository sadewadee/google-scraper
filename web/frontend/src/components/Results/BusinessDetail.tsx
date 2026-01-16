import type { ResultEntry, Review } from "../../api/types"
import {
    Chip,
    Button,
    Typography,
    Grid,
    Stack,
    Box,
    Avatar,
    Paper
} from "@mui/material"
import {
    Place as MapPin,
    Phone,
    Mail,
    Language as Globe,
    Star,
    AccessTime as Clock,
    CalendarMonth as Calendar,
    Image as ImageIcon
} from "@mui/icons-material"

interface BusinessDetailProps {
    data: ResultEntry
}

export function BusinessDetail({ data }: BusinessDetailProps) {
    const formatAddress = () => {
        if (data.complete_address) {
            const { street, city, state, postal_code, country } = data.complete_address
            return [street, city, state, postal_code, country].filter(Boolean).join(", ")
        }
        return data.address
    }

    const isOpen = () => {
        if (!data.open_hours) return null

        // Simple check logic could go here, but usually requires complex day/time parsing
        // For now, we'll just display the status string if available or generic
        return data.status === "OPEN" ?
            <Chip label="Open Now" color="success" size="small" /> :
            data.status === "CLOSED" ?
            <Chip label="Closed" color="error" size="small" /> :
            null
    }

    return (
        <Stack spacing={4} pb={4}>
            {/* Hero Section */}
            <Stack spacing={2}>
                <Box display="flex" justifyContent="space-between" alignItems="flex-start">
                    <Box>
                        <Typography variant="h5" component="h2" fontWeight="bold" gutterBottom>
                            {data.title}
                        </Typography>
                        <Stack direction="row" spacing={1} alignItems="center">
                            <Chip
                                label={data.category}
                                size="small"
                                variant="outlined"
                                sx={{ textTransform: 'uppercase', fontSize: '0.7rem' }}
                            />
                            {data.review_rating > 0 && (
                                <Box display="flex" alignItems="center" color="warning.main" gap={0.5}>
                                    <Star sx={{ fontSize: 16 }} />
                                    <Typography variant="body2" fontWeight="medium">
                                        {data.review_rating}
                                    </Typography>
                                    <Typography variant="caption" color="text.secondary">
                                        ({data.review_count})
                                    </Typography>
                                </Box>
                            )}
                        </Stack>
                    </Box>
                    {isOpen()}
                </Box>

                {/* Quick Actions */}
                <Stack direction="row" spacing={1} flexWrap="wrap">
                    {data.web_site && (
                        <Button
                            variant="outlined"
                            startIcon={<Globe sx={{ fontSize: 16 }} />}
                            href={data.web_site}
                            target="_blank"
                            rel="noopener noreferrer"
                            sx={{ flex: 1 }}
                        >
                            Website
                        </Button>
                    )}
                    <Button
                        variant="outlined"
                        startIcon={<MapPin sx={{ fontSize: 16 }} />}
                        href={`https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(data.title + " " + data.address)}&query_place_id=${data.place_id}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        sx={{ flex: 1 }}
                    >
                        Directions
                    </Button>
                    {data.phone && (
                        <Button
                            variant="outlined"
                            startIcon={<Phone sx={{ fontSize: 16 }} />}
                            href={`tel:${data.phone.replace(/\D/g, '')}`}
                            sx={{ flex: 1 }}
                        >
                            Call
                        </Button>
                    )}
                </Stack>
            </Stack>

            {/* Images */}
            {data.images && data.images.length > 0 && (
                <Stack spacing={2}>
                    <Typography variant="subtitle1" fontWeight="bold" display="flex" alignItems="center" gap={1}>
                        <ImageIcon sx={{ fontSize: 18 }} /> Photos
                    </Typography>
                    <Box
                        sx={{
                            display: 'flex',
                            gap: 2,
                            overflowX: 'auto',
                            pb: 2,
                            '&::-webkit-scrollbar': { height: 8 },
                            '&::-webkit-scrollbar-thumb': { borderRadius: 4, bgcolor: 'rgba(0,0,0,0.1)' }
                        }}
                    >
                        {data.images.map((img, idx) => (
                            <Box
                                key={idx}
                                component="img"
                                src={img.image}
                                alt={img.title || data.title}
                                sx={{
                                    height: 128,
                                    width: 192,
                                    objectFit: 'cover',
                                    borderRadius: 2,
                                    flexShrink: 0
                                }}
                                loading="lazy"
                            />
                        ))}
                    </Box>
                </Stack>
            )}

            {/* Info Cards */}
            <Grid container spacing={2}>
                <Grid size={{ xs: 12, sm: 6 }}>
                    <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
                        <Typography variant="overline" color="text.secondary" fontWeight="bold" gutterBottom display="block">
                            Contact Info
                        </Typography>

                        <Stack spacing={2}>
                            {data.address && (
                                <Box display="flex" gap={1.5} alignItems="flex-start">
                                    <MapPin sx={{ fontSize: 16, mt: 0.5, flexShrink: 0 }} />
                                    <Typography variant="body2">{formatAddress()}</Typography>
                                </Box>
                            )}

                            {data.phone && (
                                <Box display="flex" gap={1.5} alignItems="center">
                                    <Phone sx={{ fontSize: 16, flexShrink: 0 }} />
                                    <Typography variant="body2">{data.phone}</Typography>
                                </Box>
                            )}

                            {data.emails && data.emails.length > 0 && (
                                <Box display="flex" gap={1.5} alignItems="flex-start">
                                    <Mail sx={{ fontSize: 16, mt: 0.5, flexShrink: 0 }} />
                                    <Box>
                                        {data.emails.map((email, i) => (
                                            <Typography
                                                key={i}
                                                variant="body2"
                                                component="a"
                                                href={`mailto:${email}`}
                                                sx={{ display: 'block', color: 'primary.main', textDecoration: 'none', '&:hover': { textDecoration: 'underline' } }}
                                            >
                                                {email}
                                            </Typography>
                                        ))}
                                    </Box>
                                </Box>
                            )}
                        </Stack>
                    </Paper>
                </Grid>

                <Grid size={{ xs: 12, sm: 6 }}>
                    <Paper variant="outlined" sx={{ p: 2, height: '100%' }}>
                        <Typography variant="overline" color="text.secondary" fontWeight="bold" gutterBottom display="block">
                            Details
                        </Typography>

                        <Stack spacing={2}>
                            {data.plus_code && (
                                <Box display="flex" gap={1.5} alignItems="center">
                                    <Chip label="PLUS" size="small" sx={{ height: 20, fontSize: '0.65rem' }} />
                                    <Typography variant="body2">{data.plus_code}</Typography>
                                </Box>
                            )}

                            {data.price_range && (
                                <Box display="flex" gap={1.5} alignItems="center">
                                    <Typography variant="body2" fontWeight="medium">Price:</Typography>
                                    <Typography variant="body2">{data.price_range}</Typography>
                                </Box>
                            )}

                            {data.timezone && (
                                <Box display="flex" gap={1.5} alignItems="center">
                                    <Clock sx={{ fontSize: 16, flexShrink: 0 }} />
                                    <Typography variant="body2">{data.timezone}</Typography>
                                </Box>
                            )}
                        </Stack>
                    </Paper>
                </Grid>
            </Grid>

            {/* Hours */}
            {data.open_hours && Object.keys(data.open_hours).length > 0 && (
                <Paper variant="outlined" sx={{ p: 2 }}>
                    <Typography variant="overline" color="text.secondary" fontWeight="bold" gutterBottom display="flex" alignItems="center" gap={1}>
                        <Calendar sx={{ fontSize: 16 }} /> Operating Hours
                    </Typography>
                    <Stack spacing={1}>
                        {Object.entries(data.open_hours).map(([day, hours]) => (
                            <Box key={day} display="flex" justifyContent="space-between" py={0.5} borderBottom="1px solid" borderColor="divider" sx={{ '&:last-child': { borderBottom: 0 } }}>
                                <Typography variant="body2" fontWeight="medium" width={100}>{day}</Typography>
                                <Typography variant="body2" color="text.secondary" sx={{ flex: 1, textAlign: 'right' }}>
                                    {Array.isArray(hours) ? hours.join(", ") : hours}
                                </Typography>
                            </Box>
                        ))}
                    </Stack>
                </Paper>
            )}

            {/* Reviews */}
            {(data.user_reviews?.length > 0 || (data.user_reviews_extended && data.user_reviews_extended.length > 0)) && (
                <Stack spacing={2}>
                    <Typography variant="h6">Reviews</Typography>
                    <Stack spacing={2}>
                        {(data.user_reviews_extended || data.user_reviews).map((review: Review, idx: number) => (
                            <Paper key={idx} variant="outlined" sx={{ p: 2 }}>
                                <Stack spacing={2}>
                                    <Box display="flex" justifyContent="space-between" alignItems="flex-start">
                                        <Box display="flex" gap={2} alignItems="center">
                                            <Avatar src={review.profile_picture} alt={review.name}>
                                                {review.name.charAt(0)}
                                            </Avatar>
                                            <Box>
                                                <Typography variant="subtitle2">{review.name}</Typography>
                                                <Typography variant="caption" color="text.secondary">{review.when}</Typography>
                                            </Box>
                                        </Box>
                                        <Box display="flex" color="warning.main">
                                            {Array.from({ length: 5 }).map((_, i) => (
                                                <Star
                                                    key={i}
                                                    sx={{
                                                        fontSize: 14,
                                                        color: i < review.rating ? 'inherit' : '#e0e0e0'
                                                    }}
                                                />
                                            ))}
                                        </Box>
                                    </Box>
                                    {review.description && (
                                        <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic' }}>
                                            "{review.description}"
                                        </Typography>
                                    )}
                                    {review.images && review.images.length > 0 && (
                                        <Box display="flex" gap={1} overflow="auto" pt={1}>
                                            {review.images.map((img, i) => (
                                                <Box
                                                    key={i}
                                                    component="img"
                                                    src={img}
                                                    alt="Review"
                                                    sx={{ height: 64, width: 64, objectFit: 'cover', borderRadius: 1 }}
                                                />
                                            ))}
                                        </Box>
                                    )}
                                </Stack>
                            </Paper>
                        ))}
                    </Stack>
                </Stack>
            )}
        </Stack>
    )
}
