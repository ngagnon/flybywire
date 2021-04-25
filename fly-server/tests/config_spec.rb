require 'open3'

RSpec.describe 'config' do
    it 'will not start with non-existent directory' do
        stdout, status = Open3.capture2('../fly-server', '/blah/blah/blah')
        expect(status.exitstatus).to eq(1)
        expect(stdout).to include('ERROR: root directory not found: /blah/blah/blah')
    end
end