import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { organizationsListOrganizations, organizationsCreateOrganization } from '../../api-client/sdk.gen'
import type { OrganizationsOrganization as Organization } from '../../api-client'
import { CreateOrganizationModal } from './CreateOrganizationModal'

// Home page component - displays list of organizations
const HomePage = () => {
  const [organizations, setOrganizations] = useState<Organization[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isModalOpen, setIsModalOpen] = useState(false)

  useEffect(() => {
    const fetchOrganizations = async () => {
      try {
        setLoading(true)
        setError(null)
        
        const response = await organizationsListOrganizations()
        
        if (response.data) {
          setOrganizations(response.data.organizations || [])
        } else {
          throw new Error('No data received from API')
        }
      } catch (err) {
        console.error('Error fetching organizations:', err)
        setError('Failed to fetch organizations. Please try again later.')
      } finally {
        setLoading(false)
      }
    }
    
    fetchOrganizations()
  }, [])

  const handleCreateOrganization = async (name: string) => {
    const newOrg: Organization = {
      metadata: {
        name: name,
        displayName: name,
      }
    }

    const response = await organizationsCreateOrganization({ body: { organization: newOrg } })
    const organization = response.data?.organization

    if (organization) {
      setOrganizations(prev => [...prev, organization])
    }
  }

  return (
    <>
      <div className="flex justify-center w-full text-center">
        <div className="w-full max-w-6xl px-4 py-8">
          <div className="flex justify-center items-center mb-6">
            <h1 className="text-3xl font-bold">My Organizations</h1>
          </div>
        
        {loading ? (
          <div className="flex justify-center items-center h-40">
            <p className="text-gray-500">Loading organizations...</p>
          </div>
        ) : error ? (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
            <p>{error}</p>
          </div>
        ) : organizations.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-gray-500 mb-4">No organizations found</p>
            <button 
              onClick={() => setIsModalOpen(true)}
              className="flex items-center px-5 py-2.5 text-sm font-medium bg-indigo-600 text-black rounded-md hover:bg-indigo-700 transition-colors shadow-sm mx-auto"
            >
              <svg className="mr-2 h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              Create Your First Organization
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {organizations.map((organization) => (
              <div key={organization.metadata!.id} className="border rounded-lg p-5 hover:shadow-md transition-shadow flex flex-col">
                <h2 className="text-xl font-semibold mb-3">{organization.metadata!.name}</h2>
                <div className="mt-auto">
                  <Link 
                    to={`organization/${organization.metadata!.id}`}
                    className="inline-flex items-center text-sm font-medium text-indigo-600 hover:text-indigo-800 transition-colors group"
                  >
                    <span>Open Organization</span>
                    <svg className="ml-1 h-4 w-4 transform group-hover:translate-x-0.5 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 5l7 7m0 0l-7 7m7-7H3" />
                    </svg>
                  </Link>
                </div>
              </div>
            ))}
          </div>
        )}
        </div>
      </div>

      <CreateOrganizationModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSubmit={handleCreateOrganization}
      />
    </>
  )
}

export default HomePage