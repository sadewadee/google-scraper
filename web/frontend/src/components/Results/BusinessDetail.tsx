import { MapPin, Phone, Mail, Globe, Star, Clock, Calendar, Image as ImageIcon } from "lucide-react"
import type { ResultEntry, Review } from "../../api/types"
import { Button } from "../UI/Button"
import { Badge } from "../UI/Badge"

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
            <Badge variant="completed">Open Now</Badge> :
            data.status === "CLOSED" ?
            <Badge variant="failed">Closed</Badge> :
            null
    }

    return (
        <div className="space-y-6 pb-10">
            {/* Hero Section */}
            <div className="space-y-4">
                <div className="flex items-start justify-between">
                    <div>
                        <h2 className="text-2xl font-bold text-gray-800">{data.title}</h2>
                        <div className="flex items-center gap-2 mt-2 text-sm text-gray-600">
                            <span className="bg-neu-base shadow-neu-flat px-2 py-0.5 rounded text-xs font-medium uppercase tracking-wide">
                                {data.category}
                            </span>
                            {data.review_rating > 0 && (
                                <div className="flex items-center gap-1 text-yellow-600 font-medium">
                                    <Star className="w-4 h-4 fill-current" />
                                    <span>{data.review_rating}</span>
                                    <span className="text-gray-400">({data.review_count})</span>
                                </div>
                            )}
                        </div>
                    </div>
                    {isOpen()}
                </div>

                {/* Quick Actions */}
                <div className="flex flex-wrap gap-2">
                    {data.web_site && (
                        <a href={data.web_site} target="_blank" rel="noopener noreferrer" className="flex-1">
                            <Button className="w-full flex items-center justify-center gap-2 text-sm">
                                <Globe className="w-4 h-4" /> Website
                            </Button>
                        </a>
                    )}
                    <a
                        href={`https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(data.title + " " + data.address)}&query_place_id=${data.place_id}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex-1"
                    >
                        <Button className="w-full flex items-center justify-center gap-2 text-sm">
                            <MapPin className="w-4 h-4" /> Directions
                        </Button>
                    </a>
                    {data.phone && (
                        <a href={`tel:${data.phone.replace(/\D/g, '')}`} className="flex-1">
                            <Button className="w-full flex items-center justify-center gap-2 text-sm">
                                <Phone className="w-4 h-4" /> Call
                            </Button>
                        </a>
                    )}
                </div>
            </div>

            {/* Images */}
            {data.images && data.images.length > 0 && (
                <div className="space-y-3">
                    <h3 className="font-semibold text-gray-700 flex items-center gap-2">
                        <ImageIcon className="w-4 h-4" /> Photos
                    </h3>
                    <div className="flex gap-3 overflow-x-auto pb-4 custom-scrollbar snap-x">
                        {data.images.map((img, idx) => (
                            <div key={idx} className="shrink-0 snap-center">
                                <img
                                    src={img.image}
                                    alt={img.title || data.title}
                                    className="h-32 w-48 object-cover rounded-lg shadow-neu-flat"
                                    loading="lazy"
                                />
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {/* Info Cards */}
            <div className="grid gap-4 sm:grid-cols-2">
                <div className="bg-neu-base shadow-neu-flat rounded-xl p-4 space-y-3">
                    <h3 className="font-semibold text-gray-700 text-sm uppercase tracking-wider">Contact Info</h3>

                    {data.address && (
                        <div className="flex items-start gap-3 text-sm text-gray-600">
                            <MapPin className="w-4 h-4 mt-0.5 shrink-0" />
                            <span>{formatAddress()}</span>
                        </div>
                    )}

                    {data.phone && (
                        <div className="flex items-center gap-3 text-sm text-gray-600">
                            <Phone className="w-4 h-4 shrink-0" />
                            <span>{data.phone}</span>
                        </div>
                    )}

                    {data.emails && data.emails.length > 0 && (
                        <div className="flex items-start gap-3 text-sm text-gray-600">
                            <Mail className="w-4 h-4 mt-0.5 shrink-0" />
                            <div className="flex flex-col">
                                {data.emails.map((email, i) => (
                                    <a key={i} href={`mailto:${email}`} className="hover:text-primary transition-colors">
                                        {email}
                                    </a>
                                ))}
                            </div>
                        </div>
                    )}
                </div>

                <div className="bg-neu-base shadow-neu-flat rounded-xl p-4 space-y-3">
                    <h3 className="font-semibold text-gray-700 text-sm uppercase tracking-wider">Details</h3>

                    {data.plus_code && (
                        <div className="flex items-center gap-3 text-sm text-gray-600">
                            <span className="font-medium text-xs bg-gray-200 px-1.5 py-0.5 rounded">PLUS</span>
                            <span>{data.plus_code}</span>
                        </div>
                    )}

                    {data.price_range && (
                        <div className="flex items-center gap-3 text-sm text-gray-600">
                            <span className="font-medium">Price:</span>
                            <span>{data.price_range}</span>
                        </div>
                    )}

                    {data.timezone && (
                        <div className="flex items-center gap-3 text-sm text-gray-600">
                            <Clock className="w-4 h-4 shrink-0" />
                            <span>{data.timezone}</span>
                        </div>
                    )}
                </div>
            </div>

            {/* Hours */}
            {data.open_hours && Object.keys(data.open_hours).length > 0 && (
                <div className="bg-neu-base shadow-neu-flat rounded-xl p-4 space-y-3">
                    <h3 className="font-semibold text-gray-700 text-sm uppercase tracking-wider flex items-center gap-2">
                        <Calendar className="w-4 h-4" /> Operating Hours
                    </h3>
                    <div className="space-y-2">
                        {Object.entries(data.open_hours).map(([day, hours]) => (
                            <div key={day} className="flex justify-between text-sm py-1 border-b border-gray-100 last:border-0">
                                <span className="font-medium text-gray-700 w-24">{day}</span>
                                <div className="text-right text-gray-600 flex-1">
                                    {Array.isArray(hours) ? hours.join(", ") : hours}
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {/* Reviews */}
            {(data.user_reviews?.length > 0 || (data.user_reviews_extended && data.user_reviews_extended.length > 0)) && (
                <div className="space-y-4">
                    <h3 className="font-semibold text-gray-700 text-lg">Reviews</h3>
                    <div className="space-y-4">
                        {(data.user_reviews_extended || data.user_reviews).map((review: Review, idx: number) => (
                            <div key={idx} className="bg-neu-base shadow-neu-flat rounded-xl p-4 space-y-3">
                                <div className="flex items-start justify-between">
                                    <div className="flex items-center gap-3">
                                        {review.profile_picture ? (
                                            <img src={review.profile_picture} alt={review.name} className="w-10 h-10 rounded-full bg-gray-200" />
                                        ) : (
                                            <div className="w-10 h-10 rounded-full bg-gray-200 flex items-center justify-center font-bold text-gray-500">
                                                {review.name.charAt(0)}
                                            </div>
                                        )}
                                        <div>
                                            <div className="font-medium text-sm text-gray-900">{review.name}</div>
                                            <div className="text-xs text-gray-500">{review.when}</div>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-0.5 text-yellow-500">
                                        {Array.from({ length: 5 }).map((_, i) => (
                                            <Star
                                                key={i}
                                                className={`w-3 h-3 ${i < review.rating ? "fill-current" : "text-gray-300 fill-gray-300"}`}
                                            />
                                        ))}
                                    </div>
                                </div>
                                {review.description && (
                                    <p className="text-sm text-gray-600 leading-relaxed italic">
                                        "{review.description}"
                                    </p>
                                )}
                                {review.images && review.images.length > 0 && (
                                    <div className="flex gap-2 overflow-x-auto pt-2">
                                        {review.images.map((img, i) => (
                                            <img key={i} src={img} alt="Review" className="h-16 w-16 object-cover rounded shadow-sm" />
                                        ))}
                                    </div>
                                )}
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {/* Raw Metadata Debug (Optional - good for development) */}
            {/* <div className="mt-8 pt-4 border-t border-gray-200">
                <details className="text-xs text-gray-500">
                    <summary className="cursor-pointer mb-2">Raw Data</summary>
                    <pre className="overflow-auto bg-gray-50 p-2 rounded">{JSON.stringify(data, null, 2)}</pre>
                </details>
            </div> */}
        </div>
    )
}
