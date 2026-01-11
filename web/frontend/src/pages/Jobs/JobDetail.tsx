import { useParams, Link } from "react-router-dom"
import { Button } from "@/components/UI/Button"
import { Badge } from "@/components/UI/Badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/UI/Card"
import { ArrowLeft, Download, Clock, MapPin } from "lucide-react"

import { useQuery } from "@tanstack/react-query"
import { jobsApi } from "@/api/jobs"

export default function JobDetail() {
    const { id } = useParams()

    const { data: jobRes, isLoading } = useQuery({
        queryKey: ["job", id],
        queryFn: () => jobsApi.getOne(id!),
        enabled: !!id
    })

    const job = jobRes?.data

    if (isLoading) return <div className="p-8 text-center">Loading job details...</div>
    if (!job) return <div className="p-8 text-center">Job not found.</div>

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <Button variant="ghost" size="icon" asChild>
                        <Link to="/jobs">
                            <ArrowLeft className="h-4 w-4" />
                        </Link>
                    </Button>
                    <div>
                        <h2 className="text-2xl font-bold tracking-tight flex items-center gap-3">
                            Job #{id}
                            <Badge variant={job.status as any}>{job.status}</Badge>
                        </h2>
                        <p className="text-muted-foreground">Created on {new Date(job.created_at).toLocaleDateString()}</p>
                    </div>
                </div>
                <div className="flex gap-2">
                    <Button variant="outline">
                        <Download className="mr-2 h-4 w-4" />
                        Export CSV
                    </Button>
                    <Button variant="outline">
                        <Download className="mr-2 h-4 w-4" />
                        Export JSON
                    </Button>
                </div>
            </div>

            {/* Stats and Config */}
            <div className="grid gap-4 md:grid-cols-3">
                <Card>
                    <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Total Results</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">{job.result_count}</div>
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Duration</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold flex items-center gap-2">
                            <Clock className="h-4 w-4 text-muted-foreground" />
                            -
                        </div>
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Configuration</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="text-sm space-y-1">
                            <div className="flex justify-between">
                                <span className="text-muted-foreground">Keyword:</span>
                                <span className="font-medium">{job.keyword}</span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-muted-foreground">Location:</span>
                                <span className="font-medium flex items-center gap-1">
                                    <MapPin className="h-3 w-3" /> -
                                </span>
                            </div>
                        </div>
                    </CardContent>
                </Card>
            </div>

            {/* Results Table Placeholder */}
            <Card>
                <CardHeader>
                    <CardTitle>Scraped Results</CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="rounded-md border p-8 text-center text-muted-foreground">
                        Results table will be rendered here.
                    </div>
                </CardContent>
            </Card>
        </div>
    )
}
