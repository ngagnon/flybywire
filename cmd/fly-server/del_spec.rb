require 'securerandom'

RSpec.describe 'DEL' do
    context 'file' do
        ['admin', 'regular user', 'single user'].each do |persona|
            context "as #{persona}" do
                before(:all) do
                    @session = as(persona)
                    @file_name = "del-#{SecureRandom.hex}.txt"
                    @session.write_file(@file_name, "hello\nworld\n")
                    @resp = @session.cmd('DEL', @file_name)
                end

                it 'returns OK' do
                    expect(@resp).to be_ok
                end

                it 'deletes file' do
                    resp = @session.cmd('LIST', @file_name)
                    expect(resp).to be_error('NOTFOUND')
                end
            end
        end
    end

    context 'folder' do
        ['admin', 'regular user', 'single user'].each do |persona|
            context "as #{persona}" do
                before(:all) do
                    @session = as(persona)

                    @folder_name = "del-#{SecureRandom.hex}"
                    @session.cmd!('MKDIR', @folder_name)

                    @file_name = @folder_name + "/file.txt"
                    @session.write_file(@file_name, "hello\nworld\n")

                    @resp = @session.cmd('DEL', @folder_name)
                end

                it 'returns OK' do
                    expect(@resp).to be_ok
                end

                it 'deletes folder' do
                    resp = @session.cmd('LIST', "/" + @folder_name)
                    expect(resp).to be_error('NOTFOUND')
                end
            end
        end
    end
end