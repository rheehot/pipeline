/*
 * Anchore Engine API Server
 *
 * This is the Anchore Engine API. Provides the primary external API for users of the service.
 *
 * API version: 0.1.12
 * Contact: nurmi@anchore.com
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package anchore

// ContentFilesResponse File content listings from images
type ContentFilesResponse struct {
	ImageDigest string                        `json:"imageDigest,omitempty"`
	ContentType string                        `json:"content_type,omitempty"`
	Content     []ContentFilesResponseContent `json:"content,omitempty"`
}