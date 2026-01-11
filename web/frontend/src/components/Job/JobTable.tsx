import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { jobsApi } from "@/api/jobs"
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/UI/Table"
import { Badge } from "@/components/UI/Badge"
import { Button } from "@/components/UI/Button"
import { Input } from "@/components/UI/Input"
import {
    Trash2,
    Play,
    Pause,
    Search
} from "lucide-react"

export function JobTable() {
    const [searchTerm, setSearchTerm] = useState("")
    const queryClient = useQueryClient()

    const { data: jobs, isLoading } = useQuery({
        queryKey: ["jobs"],
        queryFn: jobsApi.getAll,
    })

    const deleteMutation = useMutation({
        mutationFn: jobsApi.delete,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["jobs"] })
        }
    })

    const filteredJobs = jobs?.data?.filter(job =>
        job.keyword.toLowerCase().includes(searchTerm.toLowerCase())
    ) || []

    return (
        <div className="space-y-4">
            <div className="flex items-center gap-2 max-w-sm">
                <Search className="h-4 w-4 text-muted-foreground" />
                <Input
                    placeholder="Search jobs..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="max-w-sm"
                />
            </div>

            <div className="rounded-md bg-neu-base shadow-neu-flat">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead className="w-[100px]">ID</TableHead>
                            <TableHead>Keyword</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Priority</TableHead>
                            <TableHead className="text-right">Results</TableHead>
                            <TableHead>Created At</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {isLoading ? (
                            <TableRow>
                                <TableCell colSpan={7} className="h-24 text-center">
                                    Loading jobs...
                                </TableCell>
                            </TableRow>
                        ) : filteredJobs.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={7} className="h-24 text-center">
                                    No jobs found.
                                </TableCell>
                            </TableRow>
                        ) : (
                            filteredJobs.map((job) => (
                                <TableRow key={job.id}>
                                    <TableCell className="font-medium">#{job.id}</TableCell>
                                    <TableCell className="font-medium">{job.keyword}</TableCell>
                                    <TableCell>
                                        <Badge variant={job.status}>{job.status}</Badge>
                                    </TableCell>
                                    <TableCell>
                                        <span className={`text-xs capitalize ${job.priority === 'high' ? 'text-red-500 font-bold' :
                                            job.priority === 'low' ? 'text-gray-500' : 'text-foreground'
                                            }`}>
                                            {job.priority}
                                        </span>
                                    </TableCell>
                                    <TableCell className="text-right">
                                        {job.result_count !== undefined ? job.result_count : "-"}
                                    </TableCell>
                                    <TableCell className="text-muted-foreground text-xs">
                                        {new Date(job.created_at).toLocaleDateString()}
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <div className="flex justify-end gap-2">
                                            {job.status === 'processing' ? (
                                                <Button variant="ghost" size="icon" title="Pause">
                                                    <Pause className="h-4 w-4" />
                                                </Button>
                                            ) : (job.status === 'pending' || job.status === 'cancelled') ? (
                                                <Button variant="ghost" size="icon" title="Resume/Start">
                                                    <Play className="h-4 w-4" />
                                                </Button>
                                            ) : null}

                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                className="text-destructive hover:text-destructive"
                                                title="Delete"
                                                onClick={() => deleteMutation.mutate(job.id)}
                                                disabled={deleteMutation.isPending}
                                            >
                                                <Trash2 className="h-4 w-4" />
                                            </Button>
                                        </div>
                                    </TableCell>
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
            </div>

            <div className="flex items-center justify-end space-x-2 py-4">
                <Button variant="outline" size="sm" disabled>Previous</Button>
                <Button variant="outline" size="sm" disabled>Next</Button>
            </div>
        </div>
    )
}
